package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/ttacon/chalk"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"

	"github.com/wI2L/jsondiff"
)

func main() {
	provider, err := newProvider()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}

	err = provider.Run(context.Background(), "pulumi-nanovms", "0.1.0")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}

func newProvider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithResources(
			infer.Resource(&Image{}),
		).
		WithNamespace("tpjg").
		WithDisplayName("pulumi-nanovms").
		WithDescription("A provider for NanoVMs with pulumi-go-provider.").
		WithHomepage("https://www.pulumi.com").
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"pulumi-nanovms": "index",
		}).
		Build()
}

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
	a.Describe(&i.Provider, "The target cloud provider (onprem, gcp, aws, azure, oracle, openstack, vsphere, upcloud, digitalocean)")
	a.Describe(&i.Force, "If an already existing image should be deleted if it exists")
	a.Describe(&i.UseLatestKernel, "If the latest kernel should be used, download it if necessary")
}

type ImageState struct {
	ImagePath       string `pulumi:"imagePath"`
	ImageID         string `pulumi:"imageId"`
	Config          string `pulumi:"config"`
	Checksum        string `pulumi:"checksum"`
	Provider        string `pulumi:"provider"`
	UseLatestKernel bool   `pulumi:"useLatestKernel"`
}

func (i *ImageState) Annotate(a infer.Annotator) {
	fmt.Fprintf(os.Stderr, "inferrer: %v ; i: %v\n", a, i)
	a.Describe(&i.ImagePath, "The path to the built image")
	a.Describe(&i.ImageID, "The unique identifier of the built image")
	a.Describe(&i.Config, "The configuration of the built image as a JSON encoded string")
	a.Describe(&i.Checksum, "The checksum of the built image")
	a.Describe(&i.Provider, "The cloud provider of the built image")
	a.Describe(&i.UseLatestKernel, "If the latest kernel should be used, download it if necessary")
}

func (*Image) Create(ctx context.Context, req infer.CreateRequest[ImageArgs]) (infer.CreateResponse[ImageState], error) {
	var resp infer.CreateResponse[ImageState]

	if _, err := os.Stat(req.Inputs.Elf); os.IsNotExist(err) {
		return resp, fmt.Errorf("elf file with path %s not found", req.Inputs.Elf)
	}

	builder, err := createBuilder(ctx, req.Inputs)
	if err != nil {
		return resp, err
	}

	if req.DryRun { // Don't do the actual creating if in preview
		return infer.CreateResponse[ImageState]{
			ID: req.Inputs.Name,
			Output: ImageState{
				ImagePath:       req.Inputs.Name,
				ImageID:         req.Inputs.Elf,
				Config:          string(builder.configAsJson),
				Provider:        req.Inputs.Provider,
				UseLatestKernel: req.Inputs.UseLatestKernel,
			},
		}, nil
	}

	if !req.Inputs.Force {
		//TODO: seems there is no easy way to check if the image is already built for most providers, except 'onprem'
		if req.Inputs.Provider == "onprem" {
			if _, err := os.Stat(builder.config.RunConfig.ImageName); !os.IsNotExist(err) {
				return resp, fmt.Errorf("file already exists; pass force=true to override")
			}
		}
	}

	p.GetLogger(ctx).Debugf("creating image with config: %s", builder.configAsJson)

	opsContext := lepton.NewContext(builder.config)
	imagePath, err := builder.provider.BuildImage(opsContext)
	if err != nil {
		return resp, fmt.Errorf("failed to build image: %w", err)
	}

	cs, err := checksum(imagePath)
	if err != nil {
		return resp, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	p.GetLogger(ctx).Debugf("created image with checksum: %s", cs)

	p.GetLogger(ctx).Infof("Image built successfully at %s", imagePath)

	return infer.CreateResponse[ImageState]{
		ID: req.Inputs.Name,
		Output: ImageState{
			ImagePath:       req.Inputs.Name,
			ImageID:         req.Inputs.Elf,
			Config:          string(builder.configAsJson),
			Checksum:        cs,
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
	builder, err := createBuilder(ctx, req.Inputs)
	if err != nil {
		return infer.DiffResponse{}, err
	}

	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Elf != req.State.ImageID {
		diff["elf"] = p.PropertyDiff{Kind: p.Update}
	}
	if req.Inputs.Name != req.State.ImagePath {
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

func (*Image) Read(ctx context.Context, req infer.ReadRequest[ImageArgs, ImageState]) (infer.ReadResponse[ImageArgs, ImageState], error) {
	os.WriteFile("reading.txt", []byte(req.Inputs.Name), 0644)

	p.GetLogger(ctx).Infof("READING only returns the input, it does nothing, not actually reading things anyway: %v", req)

	return infer.ReadResponse[ImageArgs, ImageState](req), nil
}

func (*Image) WireDependencies(f infer.FieldSelector, args *ImageArgs, state *ImageState) {
	f.OutputField(&state.ImageID).DependsOn(f.InputField(&args.Elf))
	f.OutputField(&state.ImagePath).DependsOn(f.InputField(&args.Name))
	f.OutputField(&state.Checksum).DependsOn(f.InputField(&args.Elf))
	f.OutputField(&state.Config).DependsOn(f.InputField(&args.Config))
	f.OutputField(&state.Provider).DependsOn(f.InputField(&args.Provider))
	f.OutputField(&state.UseLatestKernel).DependsOn(f.InputField(&args.UseLatestKernel))
}

type builder struct {
	config       *types.Config
	configAsJson string
	provider     lepton.Provider
}

func createBuilder(ctx context.Context, args ImageArgs) (*builder, error) {
	config := &types.Config{}
	config.RunConfig.Accel = true
	config.RunConfig.Memory = "2G"

	if args.Config == "" {
		p.GetLogger(ctx).Warning("no config provided, using defaults")
	} else {
		err := json.Unmarshal([]byte(args.Config), config)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal config: %w", err)
		}
	}
	config.Program = args.Elf
	config.RunConfig.ImageName = imageName(args.Name)
	config.CloudConfig.ImageName = args.Name

	if config.Kernel == "" {
		version, err := getCurrentVersion(args.UseLatestKernel)
		if err != nil {
			return nil, fmt.Errorf("failed to get kernel version: %w", err)
		}
		version = setKernelVersion(version)

		config.Kernel = getKernelVersion(version)
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

func getCurrentVersion(useLatestKernel bool) (string, error) {
	var err error

	local, remote := lepton.LocalReleaseVersion, lepton.LatestReleaseVersion
	if local == "0.0" || (useLatestKernel && remote != local) {
		arch := ""
		if runtime.GOARCH == "arm64" {
			arch = "arm"
		}
		err = lepton.DownloadReleaseImages(remote, arch)
		if err != nil {
			return "", err
		}
		return remote, nil
	}

	if parseVersion(local, 4) != parseVersion(remote, 4) {
		fmt.Println(chalk.Red, "You are running an older version of Ops.", chalk.Reset)
		fmt.Println(chalk.Red, "Update: Run", chalk.Reset, chalk.Bold.TextStyle("`ops update`"))
	}

	return local, nil
}

func parseVersion(s string, width int) int64 {
	strList := strings.Split(s, ".")
	format := fmt.Sprintf("%%s%%0%ds", width)
	v := ""
	for _, value := range strList {
		v = fmt.Sprintf(format, v, value)
	}
	var result int64
	var err error
	if result, err = strconv.ParseInt(v, 10, 64); err != nil {
		fmt.Printf("Failed to parse version %s, error is: %s", v, err)
		os.Exit(1)
	}
	return result
}

func setKernelVersion(version string) string {
	if lepton.AltGOARCH != "" {
		if lepton.AltGOARCH == "arm64" {
			return version + "-arm"
		}
		return version
	}
	if runtime.GOARCH == "arm64" {
		return version + "-arm"
	}
	return version
}

func getKernelVersion(version string) string {
	return path.Join(lepton.GetOpsHome(), version, "kernel.img")
}

func checksum(path string) (string, error) {
	hash := sha256.New()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func imageName(name string) string {
	return path.Join(lepton.GetOpsHome(), "images", name)
}
