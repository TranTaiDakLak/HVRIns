// state.go — lưu/load session state sau submit để confirm OTP ở lần chạy sau.
package igcore

import (
	"encoding/json"
	"os"
)

type savedState struct {
	DeviceID       string `json:"device_id"`
	FamilyDeviceID string `json:"family_device_id"`
	WaterfallID    string `json:"waterfall_id"`
	MachineID      string `json:"x_mid"`
	CloudTrustID   string `json:"cloud_trust_token"`
	RegFlowID      string `json:"reg_flow_id"`
	RegMachineID   string `json:"machine_id"`
	Locale         string `json:"locale"`
	RegContext     string `json:"reg_context"`
	Email          string `json:"email"`
	KeyID          string `json:"key_id"`
	PubKey         string `json:"pub_key"`
	PigeonSID      string `json:"pigeon_sid"`
	ConnUUID       string `json:"conn_uuid"`
}

func saveState(path string, p *igProfile, eng *engine, addr, keyID, pubKey string) error {
	s := savedState{
		DeviceID:       p.DeviceID,
		FamilyDeviceID: p.FamilyDeviceID,
		WaterfallID:    p.WaterfallID,
		MachineID:      p.MachineID,
		CloudTrustID:   p.CloudTrustID,
		RegFlowID:      p.RegFlowID,
		RegMachineID:   p.RegMachineID,
		Locale:         p.Locale,
		RegContext:     eng.regContext,
		Email:          addr,
		KeyID:          keyID,
		PubKey:         pubKey,
		PigeonSID:      p.PigeonSID,
		ConnUUID:       p.ConnUUID,
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func loadState(path string) (*savedState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s savedState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func profileFromState(s *savedState) *igProfile {
	return &igProfile{
		DeviceID:       s.DeviceID,
		FamilyDeviceID: s.FamilyDeviceID,
		WaterfallID:    s.WaterfallID,
		MachineID:      s.MachineID,
		CloudTrustID:   s.CloudTrustID,
		RegFlowID:      s.RegFlowID,
		RegMachineID:   s.RegMachineID,
		Locale:         s.Locale,
		PigeonSID:      s.PigeonSID,
		ConnUUID:       s.ConnUUID,
		UserAgent:      igUserAgent,
	}
}
