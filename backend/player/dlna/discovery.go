package dlna

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/av1"
)

// MediaRendererDevice represents a discovered DLNA Media Renderer device
// This replaces the go-upnpcast device.MediaRenderer
type MediaRendererDevice struct {
	FriendlyName string
	URL          string // Device description URL
	ModelName    string
	rootDevice   *goupnp.RootDevice
	location     *url.URL
}

// DiscoverMediaRenderers discovers DLNA Media Renderer devices on the network
// using the industry-standard huin/goupnp library.
//
// This function replaces go-upnpcast's SearchMediaRenderers and provides
// better compatibility with devices like Rygel that only send SSDP announcements.
func DiscoverMediaRenderers(ctx context.Context, waitSec int) ([]*MediaRendererDevice, error) {
	log.Printf("[DLNA Discovery] Starting media renderer discovery (timeout: %ds)", waitSec)

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(waitSec)*time.Second)
	defer cancel()

	var allDevices []*MediaRendererDevice
	seen := make(map[string]bool) // Deduplicate by URL

	// Try AVTransport1
	log.Printf("[DLNA Discovery] Searching for AVTransport1 services...")
	av1Clients, errors1, err1 := av1.NewAVTransport1ClientsCtx(timeoutCtx)
	if err1 != nil {
		log.Printf("[DLNA Discovery] Error discovering AVTransport1 devices: %v", err1)
	}
	for _, devErr := range errors1 {
		if devErr != nil {
			log.Printf("[DLNA Discovery] Device error (AVTransport1): %v", devErr)
		}
	}
	log.Printf("[DLNA Discovery] Found %d AVTransport1 clients", len(av1Clients))

	for _, client := range av1Clients {
		if client.RootDevice == nil {
			continue
		}
		device := client.RootDevice.Device
		locationStr := client.Location.String()
		if !seen[locationStr] {
			seen[locationStr] = true
			renderer := &MediaRendererDevice{
				FriendlyName: device.FriendlyName,
				URL:          locationStr,
				ModelName:    device.ModelName,
				rootDevice:   client.RootDevice,
				location:     client.Location,
			}
			allDevices = append(allDevices, renderer)
			log.Printf("[DLNA Discovery] Found device: %s (Model: %s, URL: %s)",
				renderer.FriendlyName, renderer.ModelName, renderer.URL)
		}
	}

	// Try AVTransport2
	log.Printf("[DLNA Discovery] Searching for AVTransport2 services...")
	av2Clients, errors2, err2 := av1.NewAVTransport2ClientsCtx(timeoutCtx)
	if err2 != nil {
		log.Printf("[DLNA Discovery] Error discovering AVTransport2 devices: %v", err2)
	}
	for _, devErr := range errors2 {
		if devErr != nil {
			log.Printf("[DLNA Discovery] Device error (AVTransport2): %v", devErr)
		}
	}
	log.Printf("[DLNA Discovery] Found %d AVTransport2 clients", len(av2Clients))

	for _, client := range av2Clients {
		if client.RootDevice == nil {
			continue
		}
		device := client.RootDevice.Device
		locationStr := client.Location.String()
		if !seen[locationStr] {
			seen[locationStr] = true
			renderer := &MediaRendererDevice{
				FriendlyName: device.FriendlyName,
				URL:          locationStr,
				ModelName:    device.ModelName,
				rootDevice:   client.RootDevice,
				location:     client.Location,
			}
			allDevices = append(allDevices, renderer)
			log.Printf("[DLNA Discovery] Found device: %s (Model: %s, URL: %s)",
				renderer.FriendlyName, renderer.ModelName, renderer.URL)
		}
	}

	log.Printf("[DLNA Discovery] Discovery complete. Found %d unique device(s)", len(allDevices))
	return allDevices, nil
}

// NewAVTransportClient creates an AVTransport client for this device
// Tries AVTransport2 first, then falls back to AVTransport1
func (d *MediaRendererDevice) NewAVTransportClient() (*av1.AVTransport1, error) {
	// Try AVTransport2 first (newer devices like Rygel use this)
	clients2, err := av1.NewAVTransport2ClientsFromRootDevice(d.rootDevice, d.location)
	if err == nil && len(clients2) > 0 {
		// Convert AVTransport2 to AVTransport1 interface
		// They have compatible methods, but we need to wrap it
		return &av1.AVTransport1{ServiceClient: clients2[0].ServiceClient}, nil
	}

	// Fall back to AVTransport1
	clients1, err := av1.NewAVTransport1ClientsFromRootDevice(d.rootDevice, d.location)
	if err != nil {
		return nil, err
	}
	if len(clients1) == 0 {
		return nil, fmt.Errorf("no AVTransport service found for device %s", d.FriendlyName)
	}
	return clients1[0], nil
}

// NewRenderingControlClient creates a RenderingControl client for this device
// Tries RenderingControl2 first, then falls back to RenderingControl1
func (d *MediaRendererDevice) NewRenderingControlClient() (*av1.RenderingControl1, error) {
	// Try RenderingControl2 first (newer devices like Rygel use this)
	clients2, err := av1.NewRenderingControl2ClientsFromRootDevice(d.rootDevice, d.location)
	if err == nil && len(clients2) > 0 {
		// Convert RenderingControl2 to RenderingControl1 interface
		return &av1.RenderingControl1{ServiceClient: clients2[0].ServiceClient}, nil
	}

	// Fall back to RenderingControl1
	clients1, err := av1.NewRenderingControl1ClientsFromRootDevice(d.rootDevice, d.location)
	if err != nil {
		return nil, err
	}
	if len(clients1) == 0 {
		return nil, fmt.Errorf("no RenderingControl service found for device %s", d.FriendlyName)
	}
	return clients1[0], nil
}
