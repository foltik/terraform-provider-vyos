package vyos

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("VYOS_KEY", nil),
			},
			"cert": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"save": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Save after making changes in Vyos",
			},
			"save_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "File to save configuration. Uses config.boot by default.",
			},
			"cache": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Use cache for read operations",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vyos_config":              resourceConfig(),
			"vyos_config_block":        resourceConfigBlock(),
			"vyos_config_block_tree":   resourceConfigBlockTree(),
			"vyos_static_host_mapping": resourceStaticHostMapping(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"vyos_config": dataSourceConfig(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

type ProviderClass struct {
	schema *schema.ResourceData
	client *client.Client

	_showCacheMutex *sync.Mutex
	_showCache      *map[string]any
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := d.Get("url").(string)
	key := d.Get("key").(string)

	cert := d.Get("cert").(string)
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

	return &ProviderClass{d, c, &sync.Mutex{}, nil}, diag.Diagnostics{}
}

func (p *ProviderClass) conditionalSave(ctx context.Context) {
	save := p.schema.Get("save").(bool)
	save_file := p.schema.Get("save_file").(string)

	if save {
		if save_file == "" {
			p.client.Config.Save(ctx)
		} else {
			p.client.Config.SaveFile(ctx, save_file)
		}
	}
}

func (p *ProviderClass) ShowCached(ctx context.Context, path string) (any, error) {
	cache := p.schema.Get("cache").(bool)

	if !cache {
		return p.client.Config.Show(ctx, path)
	}

	p._showCacheMutex.Lock()
	if p._showCache == nil {
		c := *p.client
		showCache, err := c.Config.Show(ctx, "")
		if err != nil {
			return showCache, err
		}
		switch value := showCache.(type) {
		case map[string]any:
			p._showCache = &value
		default:
			return nil, errors.New("Configuration is not a map")
		}
	}
	p._showCacheMutex.Unlock()

	var val any = *p._showCache
	for _, component := range strings.Split(path, " ") {
		obj, ok := val.(map[string]any)[component]
		if !ok {
			return nil, nil
		}
		val = obj
	}
	return val, nil

}
