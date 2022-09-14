package providerStructure

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ProviderClass struct {
	Schema *schema.ResourceData
	Client *client.Client
}

func ProviderConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := d.Get("url").(string)
	key := d.Get("key").(string)

	cert := d.Get("cert").(string)

	//lint:ignore SA4006 placeholder
	c := &client.Client{}

	if cert != "" {
		return nil, diag.Errorf("TODO: Use trusted self signed certificate")
	} else {
		// Just allow self signed certificates if a trusted cert isn't specified
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		cc := &http.Client{Transport: tr, Timeout: 10 * time.Minute}
		c = client.NewWithClient(cc, url, key)
	}

	return &ProviderClass{d, c}, diag.Diagnostics{}
}

func (p *ProviderClass) ConditionalSave(ctx context.Context) {
	save := p.Schema.Get("save").(bool)
	save_file := p.Schema.Get("save_file").(string)

	if save {
		if save_file == "" {
			p.Client.Config.Save(ctx)
		} else {
			p.Client.Config.SaveFile(ctx, save_file)
		}
	}
}
