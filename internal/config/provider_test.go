package config

import "testing"

func TestResolveEndpointDeepSeekDefaults(t *testing.T) {
	ep, err := ResolveEndpoint("deepseek", func(k string) string {
		if k == "OPENAI_API_KEY" {
			return "sk-test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.Provider != "deepseek" || ep.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("%+v", ep)
	}
	if ep.Model != "deepseek-v4-flash" || ep.APIKey != "sk-test" {
		t.Fatalf("%+v", ep)
	}
}

func TestResolveEndpointCustomRequiresURL(t *testing.T) {
	_, err := ResolveEndpoint("custom", func(k string) string {
		if k == "OPENAI_API_KEY" {
			return "sk"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected error without BASE_URL")
	}
	ep, err := ResolveEndpoint("custom", func(k string) string {
		switch k {
		case "OPENAI_API_KEY":
			return "sk"
		case "OPENAI_BASE_URL":
			return "https://relay.example.com/v1/"
		case "OPENAI_MODEL":
			return "foo"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.BaseURL != "https://relay.example.com/v1" || ep.Model != "foo" {
		t.Fatalf("%+v", ep)
	}
}

func TestResolveEndpointVendorKeyFallback(t *testing.T) {
	ep, err := ResolveEndpoint("moonshot", func(k string) string {
		if k == "MOONSHOT_API_KEY" {
			return "mk-1"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.APIKey != "mk-1" {
		t.Fatalf("key=%q", ep.APIKey)
	}
}

func TestResolveEndpointOverrideWins(t *testing.T) {
	ep, err := ResolveEndpoint("openai", func(k string) string {
		switch k {
		case "OPENAI_API_KEY":
			return "sk"
		case "OPENAI_BASE_URL":
			return "https://my-proxy.example/v1"
		case "OPENAI_MODEL":
			return "gpt-test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.BaseURL != "https://my-proxy.example/v1" || ep.Model != "gpt-test" {
		t.Fatalf("%+v", ep)
	}
}

func TestInferProviderFromURL(t *testing.T) {
	ep, err := ResolveEndpoint("", func(k string) string {
		switch k {
		case "OPENAI_BASE_URL":
			return "https://api.openai.com/v1"
		case "OPENAI_API_KEY":
			return "sk"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.Provider != "openai" {
		t.Fatalf("provider=%q", ep.Provider)
	}
}

func TestRegisterProvider(t *testing.T) {
	err := RegisterProvider(Provider{
		ID: "My-Relay", Name: "测试中转",
		DefaultURL: "https://relay.test/v1", DefaultModel: "m1",
		KeyEnv: "MY_RELAY_KEY",
	})
	if err != nil {
		t.Fatal(err)
	}
	p, ok := LookupProvider("my_relay")
	if !ok || p.DefaultURL != "https://relay.test/v1" {
		t.Fatalf("lookup %+v ok=%v", p, ok)
	}
	ep, err := ResolveEndpoint("my_relay", func(k string) string {
		if k == "MY_RELAY_KEY" {
			return "rk"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.APIKey != "rk" || ep.Model != "m1" {
		t.Fatalf("%+v", ep)
	}
}

func TestListProvidersContainsCustom(t *testing.T) {
	found := false
	for _, p := range ListProviders() {
		if p.ID == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing custom")
	}
}
