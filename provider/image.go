package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/wI2L/jsondiff"
)

type Image struct{}

var _ = (infer.CustomCreate[ImageArgs, ImageState])((*Image)(nil))
var _ = (infer.CustomDelete[ImageState])((*Image)(nil))
var _ = (infer.CustomCheck[ImageArgs])((*Image)(nil))
var _ = (infer.CustomUpdate[ImageArgs, ImageState])((*Image)(nil))
var _ = (infer.CustomDiff[ImageArgs, ImageState])((*Image)(nil))
var _ = (infer.CustomRead[ImageArgs, ImageState])((*Image)(nil))
var _ = (infer.ExplicitDependencies[ImageArgs, ImageState])((*Image)(nil))
var _ = (infer.Annotated)((*Image)(nil))
var _ = (infer.Annotated)((*ImageArgs)(nil))
var _ = (infer.Annotated)((*ImageState)(nil))

func (i *Image) Annotate(a infer.Annotator) {
	a.Describe(&i, "A NanoVMs image resource for building unikernel images")
}

type ImageArgs struct {
	Name            string `pulumi:"name"`
	Elf             string `pulumi:"elf"`
	Config          string `pulumi:"config,optional"`
	Provider        string `pulumi:"provider"`
	Force           bool   `pulumi:"force,optional"`
	UseLatestKernel bool   `pulumi:"useLatestKernel,optional"`
}

func (i *ImageArgs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the image")
	a.Describe(&i.Elf, "The path to the executable file")
	a.Describe(&i.Config, "The configuration as a JSON encoded string")
	a.Describe(&i.Provider, "The target cloud provider (onprem, gcp, aws, azure, oracle, openstack, vsphere, upcloud, do)")
	a.Describe(&i.Force, "If an already existing image should be deleted if it exists")
	a.Describe(&i.UseLatestKernel, "If the latest kernel should be used, download it if necessary")
}

type ImageState struct {
	ImagePath       string `pulumi:"imagePath"`
	ImageName       string `pulumi:"imageName"`
	Config          string `pulumi:"config"`
	Provider        string `pulumi:"provider"`
	UseLatestKernel bool   `pulumi:"useLatestKernel"`
}

func (i *ImageState) Annotate(a infer.Annotator) {
	fmt.Fprintf(os.Stderr, "inferrer: %v ; i: %v\n", a, i)
	a.Describe(&i.ImagePath, "The path to the built image")
	a.Describe(&i.ImageName, "The name of the built image")
	a.Describe(&i.Config, "The configuration of the built image as a JSON encoded string")
	a.Describe(&i.Provider, "The cloud provider of the built image")
	a.Describe(&i.UseLatestKernel, "If the latest kernel should be used, download it if necessary")
}

func (*Image) Create(ctx context.Context, req infer.CreateRequest[ImageArgs]) (infer.CreateResponse[ImageState], error) {
	var resp infer.CreateResponse[ImageState]

	if _, err := os.Stat(req.Inputs.Elf); os.IsNotExist(err) {
		return resp, fmt.Errorf("elf file with path %s not found", req.Inputs.Elf)
	}

	builder, err := createBuilder(ctx, req.Inputs, true)
	if err != nil {
		return resp, err
	}

	if req.DryRun { // Don't do the actual creating if in preview
		return infer.CreateResponse[ImageState]{
			ID: req.Inputs.Name,
			Output: ImageState{
				ImagePath:       req.Inputs.Name,
				ImageName:       req.Inputs.Elf,
				Config:          string(builder.configAsJson),
				Provider:        req.Inputs.Provider,
				UseLatestKernel: req.Inputs.UseLatestKernel,
			},
		}, nil
	}

	if !req.Inputs.Force {
		//TODO: seems there is no easy way to check if the image is already built for most providers, except 'onprem'
		if req.Inputs.Provider == "onprem" {
			if _, err := os.Stat(filepath.Join(lepton.GetOpsHome(), "instances", builder.config.RunConfig.ImageName)); !os.IsNotExist(err) {
				return resp, fmt.Errorf("file already exists; pass force=true to override")
			}
		}
	}

	p.GetLogger(ctx).Debugf("Building image with config: %s", builder.configAsJson)

	opsContext := lepton.NewContext(builder.config)
	imagePath, err := builder.provider.BuildImage(opsContext)
	if err != nil {
		return resp, fmt.Errorf("failed to build image: %w", err)
	}
	p.GetLogger(ctx).Infof("Image build, local path: %v", imagePath)

	opsContext.Config().CloudConfig.ImageName = filepath.Base(imagePath)

	err = builder.provider.CreateImage(opsContext, imagePath)
	if err != nil {
		p.GetLogger(ctx).Errorf("Error trying to create image: %v", err)
		return resp, fmt.Errorf("failed to create image: %w", err)
	}

	return infer.CreateResponse[ImageState]{
		ID: req.Inputs.Name,
		Output: ImageState{
			ImagePath:       imagePath,
			ImageName:       req.Inputs.Name,
			Config:          string(builder.configAsJson),
			Provider:        req.Inputs.Provider,
			UseLatestKernel: req.Inputs.UseLatestKernel,
		},
	}, nil
}

func (*Image) Delete(ctx context.Context, req infer.DeleteRequest[ImageState]) (infer.DeleteResponse, error) {
	var resp infer.DeleteResponse

	p.GetLogger(ctx).Infof("DELETING %v with provider %s", req.State.ImagePath, req.State.Provider)

	var config types.Config

	err := json.Unmarshal([]byte(req.State.Config), &config)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	provider, err := provider.CloudProvider(req.State.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to create provider: %w", err)
	}

	opsContext := lepton.NewContext(&config)
	err = provider.DeleteImage(opsContext, req.State.ImagePath)
	if err != nil {
		p.GetLogger(ctx).Warningf("failed to delete image: %v", err)
		return resp, err
	}

	return resp, nil
}

func (*Image) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ImageArgs], error) {
	if _, ok := req.NewInputs.GetOk("name"); !ok {
		req.NewInputs = req.NewInputs.Set("name", property.New(req.Name))
	}
	args, fails, err := infer.DefaultCheck[ImageArgs](ctx, req.NewInputs)

	provider, ok := req.NewInputs.GetOk("provider")
	if !ok {
		fails = append(fails, p.CheckFailure{
			Property: "provider",
			Reason:   "provider not specified",
		})
	} else {
		switch provider.AsString() {
		case "onprem":
			break
		case "do":
			break
		default:
			fails = append(fails, p.CheckFailure{
				Property: "provider",
				Reason:   fmt.Sprintf("provider %s not supported", provider.AsString()),
			})
		}
	}

	config, ok := req.NewInputs.GetOk("config")
	if ok {
		if !config.IsString() {
			fails = append(fails, p.CheckFailure{
				Property: "config",
				Reason:   "config must be a (JSON encoded) string",
			})
		} else {
			var c types.Config
			err := json.Unmarshal([]byte(config.AsString()), &c)
			if err != nil {
				fails = append(fails, p.CheckFailure{
					Property: "config",
					Reason:   fmt.Sprintf("invalid config: %v", err),
				})
			}
		}
	} else {
		p.GetLogger(ctx).Info("empty config field, using defaults")
	}

	return infer.CheckResponse[ImageArgs]{
		Inputs:   args,
		Failures: fails,
	}, err
}

func (i *Image) Update(ctx context.Context, req infer.UpdateRequest[ImageArgs, ImageState]) (infer.UpdateResponse[ImageState], error) {
	if !req.DryRun {
		p.GetLogger(ctx).Info("Updating resource - by creating it and overwriting the image")
	}

	createRequest := infer.CreateRequest[ImageArgs]{Inputs: req.Inputs, DryRun: req.DryRun}
	res, err := i.Create(ctx, createRequest)

	resp := infer.UpdateResponse[ImageState]{Output: res.Output}
	return resp, err
}

func (*Image) Diff(ctx context.Context, req infer.DiffRequest[ImageArgs, ImageState]) (infer.DiffResponse, error) {
	builder, err := createBuilder(ctx, req.Inputs, false)
	if err != nil {
		return infer.DiffResponse{}, err
	}

	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Name != req.State.ImageName {
		diff["name"] = p.PropertyDiff{Kind: p.Update}
	}
	patch, err := jsondiff.CompareJSON([]byte(req.State.Config), []byte(builder.configAsJson))
	if err != nil {
		return infer.DiffResponse{}, err
	}
	for _, change := range patch {
		p.GetLogger(ctx).Infof("config change: %v", change)
	}
	if builder.configAsJson == req.State.Config {
		p.GetLogger(ctx).Debugf("configs are identical: %s", builder.configAsJson)
	} else if len(patch) == 0 {
		p.GetLogger(ctx).Debugf("configs are functionally identical: %s", builder.configAsJson)
	} else {
		diff["config"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{
		DeleteBeforeReplace: false,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

func (Image) Read(ctx context.Context, req infer.ReadRequest[ImageArgs, ImageState]) (infer.ReadResponse[ImageArgs, ImageState], error) {
	resp := infer.ReadResponse[ImageArgs, ImageState](req)

	var config types.Config

	err := json.Unmarshal([]byte(req.State.Config), &config)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	provider, err := provider.CloudProvider(req.State.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to create provider: %w", err)
	}

	opsContext := lepton.NewContext(&config)
	images, err := provider.GetImages(opsContext, "")
	if err != nil {
		return resp, fmt.Errorf("failed to list images: %w", err)
	}

	if len(images) == 0 {
		p.GetLogger(ctx).Errorf("no images found")
	}

	found := false
	for _, image := range images {
		if req.State.ImageName == image.Name {
			p.GetLogger(ctx).Debugf("image %v found", image.Name)
			found = true
		}
		p.GetLogger(ctx).Debugf("image: %v ; %v ; %v ; %v", image.ID, image.Name, image.Path, image)
	}

	if !found {
		p.GetLogger(ctx).Errorf("image with name %v not found", req.State.ImageName)
		resp.ID = ""
		resp.State.ImageName = ""
	}

	return resp, nil
}

func (*Image) WireDependencies(f infer.FieldSelector, args *ImageArgs, state *ImageState) {
	f.OutputField(&state.ImageName).DependsOn(f.InputField(&args.Elf))
	f.OutputField(&state.ImagePath).DependsOn(f.InputField(&args.Name))
	f.OutputField(&state.Config).DependsOn(f.InputField(&args.Config))
	f.OutputField(&state.Provider).DependsOn(f.InputField(&args.Provider))
	f.OutputField(&state.UseLatestKernel).DependsOn(f.InputField(&args.UseLatestKernel))
}

type builder struct {
	config       *types.Config
	configAsJson string
	provider     lepton.Provider
}

func createBuilder(ctx context.Context, args ImageArgs, building bool) (*builder, error) {
	config := lepton.NewConfig()

	if args.Config == "" {
		p.GetLogger(ctx).Warning("no config provided, using defaults")
	} else {
		err := json.Unmarshal([]byte(args.Config), config)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal config: %w", err)
		}
	}

	// Note that the 'ops' tool sets various default values for the config when it
	// does `NewMergeConfigContainer` with the command line arguments.
	// Below sets the defaults similar to the 'ops' tool.

	config.Program = args.Elf
	config.Args = []string{filepath.Base(args.Elf)}
	config.RunConfig.ImageName = args.Name
	if args.Provider == "onprem" {
		config.RunConfig.ImageName = path.Join(lepton.GetOpsHome(), "images", args.Name)
	}
	config.CloudConfig.ImageName = args.Name

	arch := archCheck(config.Program)
	if arch != runtime.GOARCH && (arch+"64" != runtime.GOARCH) {
		if building {
			p.GetLogger(ctx).Warningf("Warning: Detected %s architecture in Elf binary, but running on %s, building image for %s", arch, runtime.GOARCH, arch)
		}
		lepton.AltGOARCH = arch
	}

	version, err := getCurrentVersion(ctx, args.UseLatestKernel, arch)
	if err != nil {
		return nil, fmt.Errorf("failed to get kernel version: %w", err)
	}

	if config.Kernel == "" {
		config.NanosVersion = version

		if strings.Contains(arch, "arm") {
			version += "-arm"
		}

		config.Kernel = getKernelVersion(version)
		if building {
			p.GetLogger(ctx).Infof("Using kernel version %s", config.Kernel)
		}
		config.RunConfig.Kernel = config.Kernel
	}
	config.UefiBoot = lepton.GetUefiBoot(version)
	if config.Boot == "" {
		bootPath := path.Join(lepton.GetOpsHome(), version, "boot.img")
		if _, err := os.Stat(bootPath); err == nil {
			config.Boot = bootPath
		}
	}

	// Ensure NanosVersion is set if not already provided in config
	if config.NanosVersion == "" {
		config.NanosVersion = lepton.LocalReleaseVersion
	}

	provider, err := provider.CloudProvider(args.Provider, &config.CloudConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	resultingConfig, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resultingconfig: %w", err)
	}
	return &builder{
		config:       config,
		configAsJson: string(resultingConfig),
		provider:     provider,
	}, nil
}
