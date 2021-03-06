package ankaCloudClient

import (
	"time"
)

type StandardResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Body    interface{} `json:"body,omitempty"`
}

type RegistryVmResponse struct {
	StandardResponse
	Body []VMListItem `json:"body"`
}

type ListVmResponse struct {
	StandardResponse
	Body []VM `json:"body"`
}

type VMListItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type StartVmResponse struct {
	StandardResponse
	Body []string `json:"body"`
}

type GroupResponse struct {
	StandardResponse
	Body []Group `json:"body"`
}

type Group struct {
	FallbackGroupId string  `json:"fallback_group_id"`
	Description     string  `json:"description"`
	Id              *string `json:"id"`
	Name            *string `json: "name"`
}

type GetNodeResponse struct {
	StandardResponse
	Body []Node `json:"body"`
}

type Node struct {
	NodeID         string      `json:"node_id"`
	NodeName       string      `json:"node_name"`
	IPAddress      string      `json:"ip_address"`
	CPU            uint        `json:"cpu_count,omitempty"`
	RAM            uint        `json:"ram,omitempty"`
	VMCount        uint        `json:"vm_count,omitempty"`
	VCPUCount      uint        `json:"vcpu_count,omitempty"`
	VRAM           uint        `json:"vram,omitempty"`
	CPUUtilization float32     `json:"cpu_util,omitempty"`
	RAMUtilization float32     `json:"ram_util,omitempty"`
	FreeDiskSpace  uint        `json:"free_disk_space,omitempty"`
	AnkaDiskUsage  uint        `json:"anka_disk_usage,omitempty"`
	DiskSize       uint        `json:"disk_size,omitempty"`
	State          string      `json:"state"`
	Capacity       uint        `json:"capacity"`
	Groups         []NodeGroup `json:"groups"`
}

type NodeGroup struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	FallBackGroupId string `json:"fallback_group_id"`
}

type GetVmResponse struct {
	StandardResponse
	Body VMStatus `json:"body"`
}

type StartVMRequest struct {
	VmID                   string  `json:"vmid"`
	Count                  *uint   `json:"count,omitempty"`
	Tag                    *string `json:"tag,omitempty"`
	Version                *uint   `json:"version,omitempty"`
	NodeID                 *string `json:"node_id,omitempty"`
	Name                   *string `json:"name_template,omitempty"`
	ControllerInstanceName string  `json:"name,omitempty"`
	Script                 *string `json:"startup_script,omitempty"`
	ScriptRunCondition     *int    `json:"startup_script_condition,omitempty"`
	Priority               *int    `json:"priority,omitempty"`
	GroupId                *string `json:"group_id,omitempty"`
	ControllerExternalID   string  `json:"external_id,omitempty"`
}

type TerminateVMRequest struct {
	InstanceID string `json:"id"`
}

type InstanceState string

const (
	StateScheduling  = "Scheduling"
	StateStarting    = "Pulling"
	StateStarted     = "Started"
	StateStopping    = "Stopping"
	StateStopped     = "Stopped"
	StateTerminating = "Terminating"
	StateTerminated  = "Terminated"
	StateError       = "Error"
)

type VM struct {
	Id       string   `json:"instance_id"`
	VmStatus VMStatus `json:"vm"`
}

type VMStatus struct {
	State         InstanceState `json:"instance_state"`
	Message       string        `json:"message,omitempty"`
	RegistryAddr  string        `json:"anka_registry"`
	SourceVMID    string        `json:"vmid"`
	Tag           *string       `json:"tag,omitempty"`
	Version       *uint         `json:"version,omitempty"`
	VMInfo        *VmInfo       `json:"vminfo,omitempty"`
	InFlightReqID *string       `json:"inflight_reqid,omitempty"`
	Ts            time.Time     `json:"ts"`
	CrTime        time.Time     `json:"cr_time"`
	Progress      float32       `json:"progress"`
	GroupId       string        `json:"group_id,omitempty"`
}

type VmInfo struct {
	Id                  string                `json:"uuid"`
	Name                string                `json:"name,omitempty"`
	Cpu                 int                   `json:"cpu_cores"`
	Ram                 string                `json:"ram,omitempty"`
	Status              string                `json:"status"`
	NodeId              string                `json:"node_id,omitempty"`
	HostIp              string                `json:"host_ip,omitempty"`
	VmIp                string                `json:"ip"`
	Template            string                `json:"image_id,omitempty"`
	Tag                 string                `json:"tag,omitempty"`
	VncPort             int                   `json:"vnc_port"`
	VncConnectionString string                `json:"vnc_connection_string,omitempty"`
	VncPassword         string                `json:"vnc_password,omitempty"`
	PortForwardingRules *[]PortForwardingRule `json:"port_forwarding,omitempty"`
}

type PortForwardingRule struct {
	VmPort   int    `json:"guest_port"`
	NodePort int    `json:"host_port"`
	Protocol string `json:"protocol"`
	Name     string `json: "name"`
}
