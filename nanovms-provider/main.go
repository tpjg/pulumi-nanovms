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
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/ttacon/chalk"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
)

func main() {
	provider, err := infer.NewProviderBuilder().
		WithResources(
			infer.Resource(Image{}),
		).
		WithNamespace("tpjg").
		WithDisplayName("pulumi-nanovms").
		WithDescription("A provider for NanoVMs with pulumi-go-provider.").
		WithHomepage("https://www.pulumi.com").
		Build()

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

type Image struct{}

func (i *Image) Annotate(a infer.Annotator) {
	a.Describe(&i, "A NanoVMs image resource for building and deploying unikernel images")
}

type ImageArgs struct {
	Name     string `pulumi:"name"`
	Elf      string `pulumi:"elf"`
	Config   string `pulumi:"config,optional"`
	Provider string `pulumi:"provider,optional"`
	Force    bool   `pulumi:"force,optional"`
}

func (i *ImageArgs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the image")
	a.Describe(&i.Elf, "The path to the executable file")
	a.Describe(&i.Config, "The configuration as a JSON encoded string")
	a.Describe(&i.Provider, "The target cloud provider (onprem, gcp, aws, azure, oracle, openstack, vsphere, upcloud, digitalocean)")
	a.Describe(&i.Force, "If an already existing image should be deleted if it exists")
}

type ImageState struct {
	ImagePath string `pulumi:"imagePath"`
	ImageID   string `pulumi:"imageId"`
	Config    string `pulumi:"config"`
	Checksum  string `pulumi:"checksum"`
}

func (i *ImageState) Annotate(a infer.Annotator) {
	fmt.Fprintf(os.Stderr, "inferrer: %v ; i: %v\n", a, i)
	a.Describe(&i.ImagePath, "The path to the built image")
	a.Describe(&i.ImageID, "The unique identifier of the built image")
	a.Describe(&i.Config, "The configuration of the built image as a JSON encoded string")
	a.Describe(&i.Checksum, "The checksum of the built image")
}

func (Image) Create(ctx context.Context, req infer.CreateRequest[ImageArgs]) (infer.CreateResponse[ImageState], error) {
	var resp infer.CreateResponse[ImageState]
	if !req.Inputs.Force {
		if _, err := os.Stat(req.Inputs.Name); !os.IsNotExist(err) {
			return resp, fmt.Errorf("file already exists; pass force=true to override")
		}
	}
	if req.DryRun { // Don't do the actual creating if in preview
		p.GetLogger(ctx).Info("Preview only returns the fake ID, does nothing")
		return infer.CreateResponse[ImageState]{ID: req.Inputs.Name}, nil
	}

	if _, err := os.Stat(req.Inputs.Elf); os.IsNotExist(err) {
		return resp, fmt.Errorf("elf file with path %s not found", req.Inputs.Elf)
	}

	config := &types.Config{}
	config.RunConfig.Accel = true
	config.RunConfig.Memory = "2G"

	if req.Inputs.Config == "" {
		p.GetLogger(ctx).Warning("no config provided, using defaults")
	} else {
		err := json.Unmarshal([]byte(req.Inputs.Config), config)
		if err != nil {
			return resp, fmt.Errorf("cannot unmarshal config: %w", err)
		}
	}
	config.Program = req.Inputs.Elf
	config.RunConfig.ImageName = path.Join(lepton.GetOpsHome(), "images", req.Inputs.Name+".img")
	config.CloudConfig.ImageName = req.Inputs.Name

	if config.Kernel == "" {
		version, err := getCurrentVersion()
		if err != nil {
			return resp, fmt.Errorf("failed to get kernel version: %w", err)
		}
		version = setKernelVersion(version)

		config.Kernel = getKernelVersion(version)
	}

	provider, err := provider.CloudProvider(req.Inputs.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	opsContext := lepton.NewContext(config)
	imagePath, err := provider.BuildImage(opsContext)
	if err != nil {
		return resp, fmt.Errorf("failed to build image: %w", err)
	}

	resultingConfig, err := json.Marshal(config)
	if err != nil {
		return resp, fmt.Errorf("failed to marshal resultingconfig: %w", err)
	}
	p.GetLogger(ctx).Debugf("creating image with config: %s", resultingConfig)

	cs, err := checksum(imagePath)
	if err != nil {
		return resp, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	p.GetLogger(ctx).Debugf("created image with checksum: %s", cs)

	p.GetLogger(ctx).Infof("Image built successfully at %s", imagePath)

	return infer.CreateResponse[ImageState]{
		ID: req.Inputs.Name,
		Output: ImageState{
			ImagePath: req.Inputs.Name,
			ImageID:   req.Inputs.Elf,
			Config:    string(resultingConfig),
			Checksum:  cs,
		},
	}, nil
}

func (Image) Delete(ctx context.Context, req infer.DeleteRequest[ImageArgs]) error {
	p.GetLogger(ctx).Infof("DELETING only returns the fake ID, does nothing: %v", req.State)

	return nil
}

func (Image) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[ImageArgs], error) {
	p.GetLogger(ctx).Infof("CHECKING only returns the fake ID, does nothing: %v", req)

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
	}

	return infer.CheckResponse[ImageArgs]{
		Inputs:   args,
		Failures: fails,
	}, err
}

func (Image) Update(ctx context.Context, req infer.UpdateRequest[ImageArgs, ImageState]) (infer.UpdateResponse[ImageState], error) {
	if req.DryRun { // Don't do the update if in preview
		p.GetLogger(ctx).Infof("Previewing UPDATE only returns the fake ID, does nothing: %v", req)
		return infer.UpdateResponse[ImageState]{}, nil
	}

	if req.Inputs.Elf != req.State.ImageID {
		p.GetLogger(ctx).Infof("Updating, Elf not the same")

		return infer.UpdateResponse[ImageState]{}, fmt.Errorf("Update not yet really implemented")
	}

	return infer.UpdateResponse[ImageState]{
		Output: ImageState{
			ImagePath: "updated path",
			ImageID:   "updated " + req.Inputs.Elf,
		},
	}, nil
}

func (Image) Diff(ctx context.Context, req infer.DiffRequest[ImageArgs, ImageState]) (infer.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Elf != req.State.ImageID {
		diff["elf"] = p.PropertyDiff{Kind: p.UpdateReplace} // completely replace
	}
	if req.Inputs.Name != req.State.ImagePath {
		diff["name"] = p.PropertyDiff{Kind: p.Update}
	}
	return infer.DiffResponse{
		DeleteBeforeReplace: true,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

func (Image) Read(ctx context.Context, req infer.ReadRequest[ImageArgs, ImageState]) (infer.ReadResponse[ImageArgs, ImageState], error) {
	p.GetLogger(ctx).Infof("READING only returns the input, it does nothing, not actually reading things anyway: %v", req)

	return infer.ReadResponse[ImageArgs, ImageState](req), nil
}

func (Image) WireDependencies(f infer.FieldSelector, args *ImageArgs, state *ImageState) {
	f.OutputField(&state.ImageID).DependsOn(f.InputField(&args.Elf))
	f.OutputField(&state.ImagePath).DependsOn(f.InputField(&args.Name))
	f.OutputField(&state.Checksum).DependsOn(f.InputField(&args.Elf))
	f.OutputField(&state.Config).DependsOn(f.InputField(&args.Config))
}

func getCurrentVersion() (string, error) {
	var err error

	local, remote := lepton.LocalReleaseVersion, lepton.LatestReleaseVersion
	if local == "0.0" {
		err = lepton.DownloadReleaseImages(remote, "")
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
