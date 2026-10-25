// Package disks reads the SMART state of every disk smartctl can see.
package disks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Exit status bits of smartctl: 1 command line, 2 device open failed, 4
// SMART command failed, 64 error log has errors. Only the first two mean no
// usable output.
const exitFatalMask = 1 | 2

// Disk is the SMART state of one device, keyed by serial.
type Disk struct {
	Serial            string  `json:"serial"`
	Model             string  `json:"model"`
	Device            string  `json:"device"`
	Protocol          string  `json:"protocol"`
	Firmware          string  `json:"firmware"`
	CapacityBytes     uint64  `json:"capacity_bytes"`
	Passed            *bool   `json:"passed"`
	TemperatureC      *int    `json:"temperature_c"`
	PowerOnHours      *int64  `json:"power_on_hours"`
	ReallocatedSector *int64  `json:"reallocated_sectors"`
	PendingSector     *int64  `json:"pending_sectors"`
	OfflineUncorrect  *int64  `json:"offline_uncorrectable"`
	ReportedUncorrect *int64  `json:"reported_uncorrect"`
	GrownDefects      *int64  `json:"grown_defects"`
	UncorrectedReads  *int64  `json:"uncorrected_read_errors"`
	UncorrectedWrites *int64  `json:"uncorrected_write_errors"`
	PercentageUsed    *int64  `json:"percentage_used"`
	MediaErrors       *int64  `json:"media_errors"`
	UnsafeShutdowns   *int64  `json:"unsafe_shutdowns"`
	Error             *string `json:"error"`
}

// Runner runs smartctl with the configured path and privilege command.
type Runner struct {
	Smartctl  string
	Privilege string
}

func (r Runner) command(ctx context.Context, args ...string) *exec.Cmd {
	if r.Privilege != "" {
		return exec.CommandContext(ctx, r.Privilege, append([]string{r.Smartctl}, args...)...)
	}
	return exec.CommandContext(ctx, r.Smartctl, args...)
}

func (r Runner) run(ctx context.Context, args ...string) ([]byte, int, error) {
	out, err := r.command(ctx, args...).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if len(out) == 0 {
			return nil, exitErr.ExitCode(), fmt.Errorf("smartctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return out, exitErr.ExitCode(), nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("smartctl %s: %w", strings.Join(args, " "), err)
	}
	return out, 0, nil
}

type scanDocument struct {
	Devices []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	} `json:"devices"`
}

type attribute struct {
	ID  int `json:"id"`
	Raw struct {
		Value int64 `json:"value"`
	} `json:"raw"`
}

type errorCounter struct {
	TotalUncorrectedErrors int64 `json:"total_uncorrected_errors"`
}

type infoDocument struct {
	Smartctl struct {
		ExitStatus int `json:"exit_status"`
		Messages   []struct {
			String   string `json:"string"`
			Severity string `json:"severity"`
		} `json:"messages"`
	} `json:"smartctl"`
	Device struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	ScsiRevision    string `json:"scsi_revision"`
	UserCapacity    struct {
		Bytes uint64 `json:"bytes"`
	} `json:"user_capacity"`
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature *struct {
		Current *int `json:"current"`
	} `json:"temperature"`
	PowerOnTime *struct {
		Hours int64 `json:"hours"`
	} `json:"power_on_time"`
	AtaSmartAttributes *struct {
		Table []attribute `json:"table"`
	} `json:"ata_smart_attributes"`
	ScsiGrownDefectList  *int64 `json:"scsi_grown_defect_list"`
	ScsiErrorCounterLog *struct {
		Read  *errorCounter `json:"read"`
		Write *errorCounter `json:"write"`
	} `json:"scsi_error_counter_log"`
	NvmeSmartHealthInformationLog *struct {
		PercentageUsed  int64 `json:"percentage_used"`
		MediaErrors     int64 `json:"media_errors"`
		UnsafeShutdowns int64 `json:"unsafe_shutdowns"`
	} `json:"nvme_smart_health_information_log"`
}

// Scan lists the device paths smartctl can see.
func (r Runner) Scan(ctx context.Context) ([]string, error) {
	out, _, err := r.run(ctx, "--scan", "--json")
	if err != nil {
		return nil, err
	}
	var doc scanDocument
	if uerr := json.Unmarshal(out, &doc); uerr != nil {
		return nil, fmt.Errorf("parse smartctl --scan: %w", uerr)
	}
	devices := make([]string, 0, len(doc.Devices))
	for _, device := range doc.Devices {
		devices = append(devices, device.Name)
	}
	return devices, nil
}

func int64Ptr(v int64) *int64 { return &v }

// Read collects one device.
func (r Runner) Read(ctx context.Context, device string) Disk {
	disk := Disk{Device: device}
	out, exit, err := r.run(ctx, "-x", "-j", device)
	if err != nil {
		msg := err.Error()
		disk.Error = &msg
		return disk
	}
	var doc infoDocument
	if uerr := json.Unmarshal(out, &doc); uerr != nil {
		msg := fmt.Sprintf("parse smartctl output for %s: %v", device, uerr)
		disk.Error = &msg
		return disk
	}
	if exit&exitFatalMask != 0 {
		msg := fmt.Sprintf("smartctl exit %d", exit)
		for _, m := range doc.Smartctl.Messages {
			if m.Severity == "error" {
				msg = m.String
				break
			}
		}
		disk.Error = &msg
		return disk
	}
	disk.Serial = doc.SerialNumber
	disk.Model = doc.ModelName
	disk.Protocol = doc.Device.Protocol
	disk.Firmware = doc.FirmwareVersion
	if disk.Firmware == "" {
		disk.Firmware = doc.ScsiRevision
	}
	disk.CapacityBytes = doc.UserCapacity.Bytes
	if doc.SmartStatus != nil {
		passed := doc.SmartStatus.Passed
		disk.Passed = &passed
	}
	if doc.Temperature != nil && doc.Temperature.Current != nil {
		disk.TemperatureC = doc.Temperature.Current
	}
	if doc.PowerOnTime != nil {
		disk.PowerOnHours = int64Ptr(doc.PowerOnTime.Hours)
	}
	if doc.AtaSmartAttributes != nil {
		for _, attr := range doc.AtaSmartAttributes.Table {
			switch attr.ID {
			case 5:
				disk.ReallocatedSector = int64Ptr(attr.Raw.Value)
			case 187:
				disk.ReportedUncorrect = int64Ptr(attr.Raw.Value)
			case 197:
				disk.PendingSector = int64Ptr(attr.Raw.Value)
			case 198:
				disk.OfflineUncorrect = int64Ptr(attr.Raw.Value)
			}
		}
	}
	disk.GrownDefects = doc.ScsiGrownDefectList
	if doc.ScsiErrorCounterLog != nil {
		if doc.ScsiErrorCounterLog.Read != nil {
			disk.UncorrectedReads = int64Ptr(doc.ScsiErrorCounterLog.Read.TotalUncorrectedErrors)
		}
		if doc.ScsiErrorCounterLog.Write != nil {
			disk.UncorrectedWrites = int64Ptr(doc.ScsiErrorCounterLog.Write.TotalUncorrectedErrors)
		}
	}
	if doc.NvmeSmartHealthInformationLog != nil {
		disk.PercentageUsed = int64Ptr(doc.NvmeSmartHealthInformationLog.PercentageUsed)
		disk.MediaErrors = int64Ptr(doc.NvmeSmartHealthInformationLog.MediaErrors)
		disk.UnsafeShutdowns = int64Ptr(doc.NvmeSmartHealthInformationLog.UnsafeShutdowns)
	}
	return disk
}

// Unreadable is a scanned device path smartctl could not identify, a stale
// device node for a pulled disk among them.
type Unreadable struct {
	Device string `json:"device"`
	Error  string `json:"error"`
}

// Collect scans and reads every device: disks with a serial sorted by serial
// then device path, and the devices that answered no serial listed apart.
func (r Runner) Collect(ctx context.Context) ([]Disk, []Unreadable, error) {
	devices, err := r.Scan(ctx)
	if err != nil {
		return nil, nil, err
	}
	disks := make([]Disk, 0, len(devices))
	unreadable := []Unreadable{}
	for _, device := range devices {
		disk := r.Read(ctx, device)
		if disk.Serial == "" {
			text := "no serial number"
			if disk.Error != nil {
				text = *disk.Error
			}
			unreadable = append(unreadable, Unreadable{Device: device, Error: text})
			continue
		}
		disks = append(disks, disk)
	}
	sort.Slice(disks, func(i, j int) bool {
		if disks[i].Serial != disks[j].Serial {
			return disks[i].Serial < disks[j].Serial
		}
		return disks[i].Device < disks[j].Device
	})
	return disks, unreadable, nil
}
