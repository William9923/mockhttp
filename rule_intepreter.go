package mockhttp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"

	"gopkg.in/yaml.v2"
)

type ResolverAdapter interface {
	LoadPolicy(ctx context.Context) error
	Resolve(ctx context.Context, req *Request) (*http.Response, error)
}

// --- File Based Adapter ---
type fileBasedResolver struct {
	dir      string
	policies []FileBasedMockPolicy
	isLoaded atomic.Bool
}

func NewFileResolverAdapter(dir string) (ResolverAdapter, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, err
	}
	return &fileBasedResolver{
		dir:      dir,
		policies: []FileBasedMockPolicy{},
	}, nil
}

func (r *fileBasedResolver) LoadPolicy(ctx context.Context) error {
	if r.isLoaded.Load() {
		return fmt.Errorf("policy loaded")
	}

	fileItems, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}

	for _, item := range fileItems {
		if item.IsDir() {
			continue
		}

		f, err := os.ReadFile(filepath.Join(r.dir, item.Name()))
		if err != nil {
			return err
		}

		var policy FileBasedMockPolicy
		err = yaml.Unmarshal(f, &policy)
		if err != nil {
			return err
		}

		r.policies = append(r.policies, policy)
	}

	r.isLoaded.Store(true)
	return nil
}

func (r *fileBasedResolver) Resolve(ctx context.Context, req *Request) (*http.Response, error) {
	return &http.Response{
		Body:       nil,
		StatusCode: http.StatusOK,
	}, nil
}

// --- Model ---
type FileBasedMockPolicy struct {
	Path      string          `yaml:"path"`
	Method    string          `yaml:"method"`
	Desc      string          `yaml:"desc"`
	Responses []MockResponses `yaml:"responses"`
}

type MockResponses struct {
	ResponseHeaders map[string]string `yaml:"response_headers"`
	Rules           []string          `yaml:"rules"`
	Delay           int               `yaml:"delay"`
	StatusCode      int               `yaml:"status_code"`
	EnableTemplate  bool              `yaml:"enable_template"`
}
