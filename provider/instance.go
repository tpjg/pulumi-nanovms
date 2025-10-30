package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/wI2L/jsondiff"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
)

type Instance struct{}

var _ = (infer.CustomCreate[InstanceArgs, InstanceState])((*Instance)(nil))
var _ = (infer.CustomDelete[InstanceState])((*Instance)(nil))
var _ = (infer.CustomDiff[InstanceArgs, InstanceState])((*Instance)(nil))
var _ = (infer.CustomRead[InstanceArgs, InstanceState])((*Instance)(nil))

func (i *Instance) Annotate(a infer.Annotator) {
	a.Describe(&i, "A NanoVMs resource for deploying unikernel images")
}

type InstanceArgs struct {
	ImageName string `pulumi:"image,optional"`
	Config    string `pulumi:"config"`
	Provider  string `pulumi:"provider"`
}

func (i *InstanceArgs) Annotate(a infer.Annotator) {
	a.Describe(&i.ImageName, "The name of the image to deploy")
	a.Describe(&i.Config, "The configuration for the instance")
	a.Describe(&i.Provider, "The provider for the instance")
}

type InstanceState struct {
	InstanceID string   `pulumi:"instanceID"`
	ImageName  string   `pulumi:"image"`
	Config     string   `pulumi:"config"`
	Provider   string   `pulumi:"provider"`
	PID        string   `pulumi:"pid"`
	Status     string   `pulumi:"status"`
	PublicIPs  []string `pulumi:"public_ips"`
	PrivateIPs []string `pulumi:"private_ips"`
}

func (i *InstanceState) Annotate(a infer.Annotator) {
	a.Describe(&i.InstanceID, "The unique identifier for the instance")
	a.Describe(&i.ImageName, "The name of the image deployed")
	a.Describe(&i.Config, "The configuration for the instance")
	a.Describe(&i.PID, "The provider instance ID")
	a.Describe(&i.Status, "The status of the instance")
	a.Describe(&i.PublicIPs, "The public IP addresses of the instance")
	a.Describe(&i.PrivateIPs, "The private IP addresses of the instance")
	a.Describe(&i.Provider, "The provider (type) for the instance")
}

func (*Instance) Create(ctx context.Context, req infer.CreateRequest[InstanceArgs]) (infer.CreateResponse[InstanceState], error) {
	var resp infer.CreateResponse[InstanceState]

	var config types.Config

	// In preview mode the Config may be empty, e.g. if it uses the result of an
	// image Create, in preview mode Pulumi does not wait for dependencies.
	if !req.DryRun {
		if err := json.Unmarshal([]byte(req.Inputs.Config), &config); err != nil {
			if req.Inputs.Config == "" {
				p.GetLogger(ctx).Warning("no config provided, using default")
			} else {
				return resp, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}
	}
	if req.Inputs.ImageName != "" {
		config.RunConfig.ImageName = req.Inputs.ImageName
	}
	if config.RunConfig.InstanceName == "" {
		config.RunConfig.InstanceName = fmt.Sprintf("%v-%v",
			strings.Split(filepath.Base(config.CloudConfig.ImageName), ".")[0],
			strconv.FormatInt(time.Now().Unix(), 10),
		)
	}
	// Pulumi makes the plugin binary (=this code) a process group and later
	// uses signals to kill it, prevent the instance from being killed.
	config.RunConfig.BackgroundDetach = true

	resp.ID = config.RunConfig.InstanceName
	resp.Output = InstanceState{
		InstanceID: config.RunConfig.InstanceName,
		ImageName:  config.CloudConfig.ImageName,
		Config:     req.Inputs.Config,
		Provider:   req.Inputs.Provider,
	}

	// If previewing and not running on-prem, return early, only for onprem a
	// check is usefull and other providers may need Config to be filled to
	// be able to initialize.
	if req.DryRun && req.Inputs.Provider != "onprem" {
		return resp, nil
	}

	provider, err := provider.CloudProvider(req.Inputs.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to create provider: %w", err)
	}
	opsContext := lepton.NewContext(&config)

	if req.Inputs.Provider == "onprem" {
		// Check that there is no instance running with the same image as that is not supported onprem.
		instances, err := provider.GetInstances(opsContext)
		if err == nil {
			for _, instance := range instances {
				if filepath.Base(instance.Image) == config.RunConfig.ImageName && strings.ToUpper(instance.Status) == "RUNNING" {
					if req.DryRun {
						p.GetLogger(ctx).Warningf("instance %s (with PID %s) is running, cannot run multiple instances with same image onprem", instance.Name, instance.ID)
						p.GetLogger(ctx).Warningf("stop instance before continuing if not created by this Pulumi stack (e.g. use 'ops instance delete %s')", instance.Name)
					} else {
						p.GetLogger(ctx).Errorf("stop instance before continuing (e.g. use 'ops instance delete %s')", instance.Name)
						return resp, fmt.Errorf("instance %s (with PID %s) is running, cannot run multiple instances with same image onprem", instance.Name, instance.ID)
					}
				}
			}
		} else {
			return resp, fmt.Errorf("cannot get running instances: %v", err)
		}
	}
	if !req.DryRun {
		if strings.Contains(config.Kernel, "arm") && strings.Contains(runtime.GOARCH, "amd") {
			// running on amd64 but starting an arm64 instance, set AltGOARCH
			lepton.AltGOARCH = "arm64"
			p.GetLogger(ctx).Infof("creating instance on %s for %s with architecture: %v", req.Inputs.Provider, config.CloudConfig.ImageName, lepton.AltGOARCH)
		} else if !strings.Contains(config.Kernel, "arm") && strings.Contains(runtime.GOARCH, "arm") {
			// running on arm64 but starting an amd64 instance, set AltGOARCH
			lepton.AltGOARCH = "amd64"
			p.GetLogger(ctx).Infof("creating instance on %s for %s with architecture: %v", req.Inputs.Provider, config.CloudConfig.ImageName, lepton.AltGOARCH)
		} else {
			p.GetLogger(ctx).Infof("creating instance on %s for %s", req.Inputs.Provider, config.CloudConfig.ImageName)
		}
		err = provider.CreateInstance(opsContext)
		if err != nil {
			return resp, fmt.Errorf("failed to create instance: %w", err)
		}
		if req.Inputs.Provider == "onprem" {
			time.Sleep(200 * time.Millisecond)
			p.GetLogger(ctx).Infof("created the instance, returning response!")
			time.Sleep(500 * time.Millisecond)
		}
	}

	if !req.DryRun {
		resp.Output.Status = "starting"
		resp.Output.PublicIPs = []string{}
		resp.Output.PrivateIPs = []string{}
	}
	return resp, nil
}

func (*Instance) Delete(ctx context.Context, req infer.DeleteRequest[InstanceState]) (infer.DeleteResponse, error) {
	resp := infer.DeleteResponse{}

	var config types.Config
	if err := json.Unmarshal([]byte(req.State.Config), &config); err != nil {
		if req.State.Config == "" {
			p.GetLogger(ctx).Info("no config provided, cannot delete instance")
			return resp, nil
		} else {
			return resp, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	provider, err := provider.CloudProvider(req.State.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to get provider: %w", err)
	}

	p.GetLogger(ctx).Infof("deleting instance %v on provider %v", req.State.InstanceID, req.State.Provider)

	opsContext := lepton.NewContext(&config)

	err = provider.DeleteInstance(opsContext, req.State.InstanceID)
	if err != nil {
		if strings.Contains(err.Error(), "instance not found") {
			p.GetLogger(ctx).Infof("instance %v not found - no longer running?", req.State.InstanceID)
		} else {
			return resp, fmt.Errorf("failed to delete instance: %w", err)
		}
	}
	return resp, nil
}

func (i *Instance) Diff(ctx context.Context, req infer.DiffRequest[InstanceArgs, InstanceState]) (infer.DiffResponse, error) {
	resp := infer.DiffResponse{}

	diffs := map[string]p.PropertyDiff{}
	var config types.Config
	if err := json.Unmarshal([]byte(req.State.Config), &config); err != nil {
		if req.State.Config == "" {
			p.GetLogger(ctx).Info("no state config provided, cannot diff instance")
			diffs["config"] = p.PropertyDiff{Kind: p.DeleteReplace}
			resp.HasChanges = true
			resp.DeleteBeforeReplace = true
			resp.DetailedDiff = diffs
			return resp, nil
		} else {
			return resp, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	var argconfig types.Config
	if err := json.Unmarshal([]byte(req.Inputs.Config), &argconfig); err != nil {
		if req.Inputs.Config == "" {
			p.GetLogger(ctx).Info("no input config provided, cannot diff instance")
			diffs["config"] = p.PropertyDiff{Kind: p.DeleteReplace}
			resp.HasChanges = true
			resp.DeleteBeforeReplace = true
			resp.DetailedDiff = diffs
			return resp, nil
		} else {
			return resp, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	patches, err := jsondiff.CompareJSON([]byte(req.State.Config), []byte(req.Inputs.Config))
	if err != nil {
		return resp, err
	}
	for _, patch := range patches {
		p.GetLogger(ctx).Infof("config patch: %s %v -> %v", patch.Path, patch.OldValue, patch.Value)
		diffs[patch.Path] = p.PropertyDiff{Kind: p.UpdateReplace}
		resp.HasChanges = true
	}

	if req.State.ImageName != req.Inputs.ImageName {
		p.GetLogger(ctx).Infof("image name changed from %s to %s", req.State.ImageName, req.Inputs.ImageName)
		diffs["image_name"] = p.PropertyDiff{Kind: p.UpdateReplace}
		resp.HasChanges = true
	}

	resp.HasChanges = resp.HasChanges || (len(diffs) > 0)
	resp.DeleteBeforeReplace = resp.HasChanges
	resp.DetailedDiff = diffs
	return resp, nil
}

func (Instance) Read(ctx context.Context, req infer.ReadRequest[InstanceArgs, InstanceState]) (infer.ReadResponse[InstanceArgs, InstanceState], error) {
	p.GetLogger(ctx).Debugf("reading instance %v information on provider %v", req.State.InstanceID, req.State.Provider)

	resp := infer.ReadResponse[InstanceArgs, InstanceState](req)

	var config types.Config
	if err := json.Unmarshal([]byte(req.State.Config), &config); err != nil {
		if req.State.Config == "" {
			p.GetLogger(ctx).Info("no config provided, cannot get instance status")
			return resp, nil
		} else {
			return resp, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	provider, err := provider.CloudProvider(req.State.Provider, &config.CloudConfig)
	if err != nil {
		return resp, fmt.Errorf("failed to get provider: %w", err)
	}

	p.GetLogger(ctx).Infof("getting instance %v information on provider %v", req.State.InstanceID, req.State.Provider)

	opsContext := lepton.NewContext(&config)

	instance, err := provider.GetInstanceByName(opsContext, req.State.InstanceID)
	if err != nil {
		if strings.Contains(err.Error(), "instance not found") {
			p.GetLogger(ctx).Infof("instance %v not found - no longer running?", req.State.InstanceID)
			resp.ID = ""
			resp.State.ImageName = ""
			return resp, nil
		} else {
			return resp, fmt.Errorf("failed to get instance information: %w", err)
		}
	}

	p.GetLogger(ctx).Infof("instance %v status: %v", instance.ID, instance.Status)
	resp.State.PID = instance.ID
	resp.State.Status = instance.Status
	resp.State.PublicIPs = instance.PublicIps
	resp.State.PrivateIPs = instance.PrivateIps

	return resp, nil
}
