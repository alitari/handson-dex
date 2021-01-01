package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/spf13/cobra"
)

const (
	bearerSchema string = "Bearer "
)

type user struct {
	email  string
	groups []string
}

type app struct {
	clientID     string
	clientSecret string

	verifier *oidc.IDTokenVerifier
	provider *oidc.Provider
	ctx      context.Context

	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope bool

	client *http.Client
}

type debugTransport struct {
	t http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	log.Printf("%s", reqDump)

	resp, err := d.t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	log.Printf("%s", respDump)
	return resp, nil
}

func cmd() *cobra.Command {
	var (
		a         app
		issuerURL string
		listen    string
		debug     bool
	)
	c := cobra.Command{
		Use:   "dex-oidconsumer",
		Short: "An example OpenID consumer client",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("surplus arguments provided")
			}

			listenURL, err := url.Parse(listen)
			if err != nil {
				return fmt.Errorf("parse listen address: %v", err)
			}

			if debug {
				if a.client == nil {
					a.client = &http.Client{
						Transport: debugTransport{http.DefaultTransport},
					}
				} else {
					a.client.Transport = debugTransport{a.client.Transport}
				}
			}

			if a.client == nil {
				a.client = http.DefaultClient
			}

			// TODO(ericchiang): Retry with backoff
			ctx := oidc.ClientContext(context.Background(), a.client)
			provider, err := oidc.NewProvider(ctx, issuerURL)
			if err != nil {
				return fmt.Errorf("failed to query provider %q: %v", issuerURL, err)
			}

			var s struct {
				// What scopes does a provider support?
				//
				// See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
				ScopesSupported []string `json:"scopes_supported"`
			}
			if err := provider.Claims(&s); err != nil {
				return fmt.Errorf("failed to parse provider scopes_supported: %v", err)
			}

			if len(s.ScopesSupported) == 0 {
				// scopes_supported is a "RECOMMENDED" discovery claim, not a required
				// one. If missing, assume that the provider follows the spec and has
				// an "offline_access" scope.
				a.offlineAsScope = true
			} else {
				// See if scopes_supported has the "offline_access" scope.
				a.offlineAsScope = func() bool {
					for _, scope := range s.ScopesSupported {
						if scope == oidc.ScopeOfflineAccess {
							return true
						}
					}
					return false
				}()
			}

			a.provider = provider
			a.ctx = ctx
			a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})

			http.HandleFunc("/", a.handleIndex)

			log.Printf("listening on %s", listen)
			return http.ListenAndServe(listenURL.Host, nil)
		},
	}
	c.Flags().StringVar(&a.clientID, "client-id", "dex-clientapp", "OAuth2 client ID of this application.")
	c.Flags().StringVar(&a.clientSecret, "client-secret", "example-secret", "OAuth2 client secret of this application.")
	c.Flags().StringVar(&issuerURL, "issuer", "http://127.0.0.1:5556/dex", "URL of the OpenID Connect issuer.")
	c.Flags().StringVar(&listen, "listen", "http://0.0.0.0:5555", "HTTP(S) address to listen at.")
	c.Flags().BoolVar(&debug, "debug", false, "Print all request and responses from the OpenID Connect issuer.")
	return &c
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	headerSchema := "header is supposed to look like 'Authorization: Bearer <token>'"
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required. "+headerSchema, http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(authHeader, bearerSchema) {
		http.Error(w, "Authorization requires Bearer scheme. "+headerSchema, http.StatusBadRequest)
		return
	}
	bearerToken := authHeader[len(bearerSchema):]
	user, err := a.authorize(bearerToken)
	if err != nil {
		http.Error(w, "Authorization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("User authenticated! email: %s, member of groups: %v", user.email, user.groups)))
}

func (a *app) authorize(bearerToken string) (*user, error) {
	idToken, err := a.verifier.Verify(a.ctx, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("could not verify bearer token: %v", err)
	}
	// Extract custom claims.
	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	if !claims.Verified {
		return nil, fmt.Errorf("email (%q) in returned claims was not verified", claims.Email)
	}
	return &user{claims.Email, claims.Groups}, nil
}
