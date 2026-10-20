package manifest_test

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

const valid = `
version = 1

[[service]]
name = "gateway"
description = "The way in"
command = ["bin/gateway", "-v"]
depends_on = ["home"]

[[service]]
name = "home"
description = "The Home screen"
command = ["bin/home"]
`

func TestParseReadsEveryField(t *testing.T) {
	services, err := manifest.Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}

	want := []manifest.Service{
		{Name: "home", Description: "The Home screen", Command: []string{"bin/home"}},
		{
			Name:        "gateway",
			Description: "The way in",
			Command:     []string{"bin/gateway", "-v"},
			DependsOn:   []string{"home"},
		},
	}
	if !reflect.DeepEqual(services, want) {
		t.Errorf("got %+v, want %+v", services, want)
	}
}

func TestParseRejectsABadManifest(t *testing.T) {
	const badName = "names use a-z, 0-9 and hyphens, and start with a letter"

	tests := []struct {
		name, doc, want string
	}{
		{
			"broken TOML",
			"version = 1\n[[service]\n",
			"line 2: expected ']]' to close array table name",
		},
		{
			"an unknown key",
			manifestOf("home") + "depends-on = [\"web\"]\n",
			"line 6: service.depends-on: unknown key",
		},
		{"an unknown table", "version = 1\n[[services]]\n", "line 2: services: unknown key"},
		{
			"a value of the wrong type",
			oneService(`command = "bin/home"`),
			"line 3: service.command: cannot decode TOML string into struct field " +
				"manifest.Service.Command of type []string",
		},
		{"no version", serviceTable("home"), "version must be 1, not 0"},
		{"another version", "version = 2\n" + serviceTable("home"), "version must be 1, not 2"},
		{"no services", "version = 1\n", "no services"},
		{"a service without a name", manifestOf("home", ""), "service 2: no name"},
		{"a name with a capital", manifestOf("Home"), `service "Home": ` + badName},
		{
			"a name that leaves its directory",
			manifestOf("../home"),
			`service "../home": ` + badName,
		},
		{"a name that starts with a digit", manifestOf("2home"), `service "2home": ` + badName},
		{
			"a blank description",
			oneService(`name = "home"`, `description = " "`, `command = ["bin/home"]`),
			`service "home": no description`,
		},
		{
			"no command",
			oneService(`name = "home"`, `description = "Home"`),
			`service "home": no command`,
		},
		{
			"an empty program",
			oneService(`name = "home"`, `description = "Home"`, `command = ["", "-v"]`),
			`service "home": no command`,
		},
		{"a name used twice", manifestOf("home", "home"), `service "home": defined twice`},
		{
			"an unknown dependency",
			manifestOf("home:wbe"),
			`service "home": unknown dependency "wbe"`,
		},
		{"a dependency on itself", manifestOf("home:home"), "dependency cycle: home → home"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			services, err := manifest.Parse([]byte(test.doc))
			if err == nil {
				t.Fatalf("got %+v, want the error %q", services, test.want)
			}

			if err.Error() != test.want {
				t.Errorf("got the error %q, want %q", err, test.want)
			}
		})
	}
}

func TestTheReposManifestIsValid(t *testing.T) {
	data, err := os.ReadFile("../../../../services.toml")
	if err != nil {
		t.Fatal(err)
	}

	services, err := manifest.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := names(services), []string{"home", "web", "gateway"}; !slices.Equal(got, want) {
		t.Errorf("start order = %q, want %q", got, want)
	}
}

// manifestOf writes a manifest with a service for each spec (see serviceTable).
func manifestOf(specs ...string) string {
	var doc strings.Builder

	doc.WriteString("version = 1\n")

	for _, spec := range specs {
		doc.WriteString(serviceTable(spec))
	}

	return doc.String()
}

// serviceTable writes a service for a "name:dependency,dependency" spec.
func serviceTable(spec string) string {
	var table strings.Builder

	name, dependencies, _ := strings.Cut(spec, ":")
	fmt.Fprintf(&table, "[[service]]\nname = %q\ndescription = \"A service\"\n", name)
	fmt.Fprintf(&table, "command = [\"bin/%s\"]\n", name)

	if dependencies != "" {
		quoted := strings.ReplaceAll(dependencies, ",", `", "`)
		fmt.Fprintf(&table, "depends_on = [\"%s\"]\n", quoted)
	}

	return table.String()
}

// oneService writes a manifest with one service made of the given lines.
func oneService(lines ...string) string {
	return "version = 1\n[[service]]\n" + strings.Join(lines, "\n") + "\n"
}

func names(services []manifest.Service) []string {
	var result []string
	for _, svc := range services {
		result = append(result, svc.Name)
	}

	return result
}
