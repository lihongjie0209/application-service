package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,127}$`)

type Manifest struct {
	Applications []ApplicationSpec `yaml:"applications"`
}

type ApplicationSpec struct {
	Code         string         `yaml:"code"`
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description"`
	Icon         string         `yaml:"icon"`
	DefaultRoute string         `yaml:"default_route"`
	SortOrder    int32          `yaml:"sort_order"`
	Metadata     map[string]any `yaml:"metadata"`
	Menus        []MenuSpec     `yaml:"menus"`
}

type MenuSpec struct {
	Code            string `yaml:"code"`
	Parent          string `yaml:"parent"`
	Type            string `yaml:"type"`
	Name            string `yaml:"name"`
	I18nKey         string `yaml:"i18n_key"`
	Route           string `yaml:"route"`
	Component       string `yaml:"component"`
	Icon            string `yaml:"icon"`
	ExternalURL     string `yaml:"external_url"`
	PermissionCode  string `yaml:"permission_code"`
	PermissionScope string `yaml:"permission_scope"`
	SortOrder       int32  `yaml:"sort_order"`
	Visible         *bool  `yaml:"visible"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read bootstrap manifest: %w", err)
	}
	var manifest Manifest
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode bootstrap manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if len(m.Applications) == 0 {
		return errors.New("bootstrap manifest requires at least one application")
	}
	applications := make(map[string]struct{}, len(m.Applications))
	for _, application := range m.Applications {
		application.Code = strings.TrimSpace(application.Code)
		if !codePattern.MatchString(application.Code) || strings.TrimSpace(application.Name) == "" || len(application.Menus) == 0 {
			return fmt.Errorf("application code, name, and menus are required")
		}
		if _, exists := applications[application.Code]; exists {
			return fmt.Errorf("duplicate application code %q", application.Code)
		}
		applications[application.Code] = struct{}{}
		menus := make(map[string]MenuSpec, len(application.Menus))
		for _, menu := range application.Menus {
			menu.Code = strings.TrimSpace(menu.Code)
			if menu.Code == "" || strings.TrimSpace(menu.Name) == "" {
				return fmt.Errorf("application %q has a menu without code or name", application.Code)
			}
			if !codePattern.MatchString(menu.Code) {
				return fmt.Errorf("application %q has invalid menu code %q", application.Code, menu.Code)
			}
			menuType := strings.TrimSpace(menu.Type)
			if menuType == "" {
				menuType = "page"
			}
			switch menuType {
			case "page":
				if strings.TrimSpace(menu.Route) == "" || strings.TrimSpace(menu.Component) == "" {
					return fmt.Errorf("application %q page menu %q requires route and component", application.Code, menu.Code)
				}
			case "directory":
				if menu.Component != "" || menu.ExternalURL != "" {
					return fmt.Errorf("application %q directory menu %q cannot define component or external_url", application.Code, menu.Code)
				}
			case "external":
				if strings.TrimSpace(menu.ExternalURL) == "" {
					return fmt.Errorf("application %q external menu %q requires external_url", application.Code, menu.Code)
				}
			default:
				return fmt.Errorf("application %q menu %q has invalid type %q", application.Code, menu.Code, menuType)
			}
			if _, exists := menus[menu.Code]; exists {
				return fmt.Errorf("application %q has duplicate menu code %q", application.Code, menu.Code)
			}
			if menu.Component != "" && !strings.HasPrefix(menu.Component, application.Code+".") {
				return fmt.Errorf("menu %q component %q does not belong to application %q", menu.Code, menu.Component, application.Code)
			}
			permissionScope := strings.ToLower(strings.TrimSpace(menu.PermissionScope))
			if permissionScope != "" && permissionScope != "tenant" && permissionScope != "platform" {
				return fmt.Errorf("application %q menu %q has invalid permission_scope %q", application.Code, menu.Code, menu.PermissionScope)
			}
			menus[menu.Code] = menu
		}
		for _, menu := range application.Menus {
			if menu.Parent != "" {
				if _, exists := menus[menu.Parent]; !exists {
					return fmt.Errorf("application %q menu %q has unknown parent %q", application.Code, menu.Code, menu.Parent)
				}
			}
		}
		if _, err := orderedMenus(application.Menus); err != nil {
			return fmt.Errorf("application %q: %w", application.Code, err)
		}
	}
	return nil
}

func orderedMenus(values []MenuSpec) ([]MenuSpec, error) {
	remaining := append([]MenuSpec(nil), values...)
	result := make([]MenuSpec, 0, len(values))
	created := make(map[string]struct{}, len(values))
	for len(remaining) > 0 {
		progress := false
		next := make([]MenuSpec, 0, len(remaining))
		for _, menu := range remaining {
			if menu.Parent == "" {
				result, created, progress = append(result, menu), addCode(created, menu.Code), true
				continue
			}
			if _, ok := created[menu.Parent]; ok {
				result, created, progress = append(result, menu), addCode(created, menu.Code), true
				continue
			}
			next = append(next, menu)
		}
		if !progress {
			return nil, errors.New("menu parent graph contains a cycle")
		}
		remaining = next
	}
	return result, nil
}

func addCode(values map[string]struct{}, code string) map[string]struct{} {
	values[code] = struct{}{}
	return values
}

func visible(value *bool) bool {
	return value == nil || *value
}
