package provider

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-provider-vault/internal/consts"
)

func TestAuthLoginOIDC_Init(t *testing.T) {
	tests := []struct {
		name         string
		authField    string
		raw          map[string]interface{}
		wantErr      bool
		expectParams map[string]interface{}
		expectErr    error
	}{
		{
			name:      "basic",
			authField: consts.FieldAuthLoginOIDC,
			raw: map[string]interface{}{
				consts.FieldAuthLoginOIDC: []interface{}{
					map[string]interface{}{
						consts.FieldNamespace: "ns1",
						consts.FieldRole:      "alice",
					},
				},
			},
			expectParams: map[string]interface{}{
				consts.FieldNamespace:               "ns1",
				consts.FieldMount:                   consts.MountTypeOIDC,
				consts.FieldRole:                    "alice",
				consts.FieldCallbackListenerAddress: "",
				consts.FieldCallbackAddress:         "",
			},
			wantErr: false,
		},
		{
			name:         "error-missing-resource",
			authField:    consts.FieldAuthLoginOIDC,
			expectParams: nil,
			wantErr:      true,
			expectErr:    fmt.Errorf("resource data missing field %q", consts.FieldAuthLoginOIDC),
		},
		{
			name:      "error-missing-required",
			authField: consts.FieldAuthLoginOIDC,
			raw: map[string]interface{}{
				consts.FieldAuthLoginOIDC: []interface{}{
					map[string]interface{}{},
				},
			},
			expectParams: nil,
			wantErr:      true,
			expectErr: fmt.Errorf("required fields are unset: %v", []string{
				consts.FieldRole,
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := map[string]*schema.Schema{
				tt.authField: GetOIDCLoginSchema(tt.authField),
			}

			d := schema.TestResourceDataRaw(t, s, tt.raw)
			l := &AuthLoginOIDC{}
			err := l.Init(d, tt.authField)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if tt.expectErr != nil {
					if !reflect.DeepEqual(tt.expectErr, err) {
						t.Errorf("Init() expected error %#v, actual %#v", tt.expectErr, err)
					}
				}
			} else {
				if !reflect.DeepEqual(tt.expectParams, l.params) {
					t.Errorf("Init() expected params %#v, actual %#v", tt.expectParams, l.params)
				}
			}
		})
	}
}

func TestAuthLoginOIDC_LoginPath(t *testing.T) {
	type fields struct {
		AuthLoginCommon AuthLoginCommon
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				AuthLoginCommon: AuthLoginCommon{
					params: map[string]interface{}{
						consts.FieldRole: "alice",
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &AuthLoginOIDC{
				AuthLoginCommon: tt.fields.AuthLoginCommon,
			}
			if got := l.LoginPath(); got != tt.want {
				t.Errorf("LoginPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthLoginOIDC_getAuthParams(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]interface{}
		want      map[string]string
		expectErr error
		wantErr   bool
	}{
		{
			name: "listener-addr-only",
			params: map[string]interface{}{
				consts.FieldRole:                    "alice",
				consts.FieldCallbackListenerAddress: "tcp://localhost:55000",
				consts.FieldCallbackAddress:         "",
			},
			want: map[string]string{
				consts.FieldMount:  consts.MountTypeOIDC,
				consts.FieldRole:   "alice",
				fieldSkipBrowser:   "true",
				fieldListenAddress: "localhost",
				fieldPort:          "55000",
			},
			wantErr: false,
		},
		{
			name: "callback-addr-only",
			params: map[string]interface{}{
				consts.FieldRole:            "alice",
				consts.FieldCallbackAddress: "http://127.0.0.1:55001",
			},
			want: map[string]string{
				consts.FieldMount:   consts.MountTypeOIDC,
				consts.FieldRole:    "alice",
				fieldSkipBrowser:    "true",
				fieldCallbackHost:   "127.0.0.1",
				fieldCallbackPort:   "55001",
				fieldCallbackMethod: "http",
			},
			wantErr: false,
		},
		{
			name: "both-addrs",
			params: map[string]interface{}{
				consts.FieldRole:                    "alice",
				consts.FieldCallbackListenerAddress: "tcp://localhost:55000",
				consts.FieldCallbackAddress:         "http://127.0.0.1:55001",
			},
			want: map[string]string{
				consts.FieldMount:   consts.MountTypeOIDC,
				consts.FieldRole:    "alice",
				fieldSkipBrowser:    "true",
				fieldListenAddress:  "localhost",
				fieldPort:           "55000",
				fieldCallbackHost:   "127.0.0.1",
				fieldCallbackPort:   "55001",
				fieldCallbackMethod: "http",
			},
			wantErr: false,
		},
		{
			name: "error-no-role",
			params: map[string]interface{}{
				consts.FieldCallbackListenerAddress: "tcp://localhost:55000",
				consts.FieldCallbackAddress:         "http://127.0.0.1:55001",
			},
			want:      nil,
			wantErr:   true,
			expectErr: fmt.Errorf("%q is not set", consts.FieldRole),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &AuthLoginOIDC{
				AuthLoginCommon: AuthLoginCommon{
					params:      tt.params,
					initialized: true,
				},
			}
			got, err := l.getAuthParams()
			if (err != nil) != tt.wantErr {
				t.Errorf("getAuthParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(err, tt.expectErr) {
				t.Errorf("getAuthParams() gotErr = %v, wantErr %v", err, tt.expectErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAuthParams() got = %v, want %v", got, tt.want)
			}
		})
	}
}
