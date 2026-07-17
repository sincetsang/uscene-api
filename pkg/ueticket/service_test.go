package ueticket

import (
	"TMA/pkg/nucl"
	"testing"
	"time"

	"github.com/sulink/ueapisdk/models"
)

func TestTmpTicketDtoUnmarshalSupportsUECasing(t *testing.T) {
	cases := []struct {
		name     string
		payload  string
		deviceID string
		expires  int
	}{
		{name: "camel", payload: `{"deviceId":"ticket-a","expiresIn":120}`, deviceID: "ticket-a", expires: 120},
		{name: "lower", payload: `{"deviceid":"ticket-b","expiresin":60}`, deviceID: "ticket-b", expires: 60},
		{name: "pascal", payload: `{"DeviceId":"ticket-c","ExpiresIn":30}`, deviceID: "ticket-c", expires: 30},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dto models.TmpTicketDto
			if err := dto.UnmarshalJSON([]byte(tc.payload)); err != nil {
				t.Fatal(err)
			}
			if dto.DeviceID != tc.deviceID || dto.ExpiresIn != tc.expires {
				t.Fatalf("unexpected dto: %+v", dto)
			}
		})
	}
}

func TestHashClientHardIDDoesNotExposeOriginal(t *testing.T) {
	hashA := hashClientHardID("device-123")
	hashB := hashClientHardID(" device-123 ")
	if hashA != hashB {
		t.Fatalf("normalized hardid should produce the same hash")
	}
	if hashA == "device-123" || len(hashA) != 64 {
		t.Fatalf("unexpected hardid hash: %s", hashA)
	}
}

func TestTicketAndFlowTTLDefaults(t *testing.T) {
	nu := &nucl.Nucleus{Config: &nucl.Config{}}
	service := New(nu)
	if service.ticketTTL() != 120*time.Second {
		t.Fatalf("unexpected default ticket TTL: %s", service.ticketTTL())
	}
	if service.flowTTL() != 5*time.Minute {
		t.Fatalf("unexpected default flow TTL: %s", service.flowTTL())
	}
	nu.Config.UE.TmpTicket.TicketTTLSeconds = 300
	if service.ticketTTL() != 120*time.Second {
		t.Fatalf("ticket TTL must not exceed 120 seconds: %s", service.ticketTTL())
	}
	nu.Config.UE.TmpTicket.TicketTTLSeconds = 60
	if service.ticketTTL() != 60*time.Second {
		t.Fatalf("shorter upstream TTL should be honored: %s", service.ticketTTL())
	}
}

func TestEndpointValidation(t *testing.T) {
	if err := validateEndpoint("https://ue.example.test/tmp-ticket", "ticket"); err != nil {
		t.Fatal(err)
	}
	if err := validateEndpoint("/tmp-ticket", "ticket"); err == nil {
		t.Fatal("relative endpoint should be rejected")
	}
}
