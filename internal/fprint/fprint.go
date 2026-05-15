package fprint

import (
	"os/exec"
	"strings"
)

type PackageManager struct {
	Name    string
	Query   []string
	Install []string
	Pkgs    []string
}

var managers = []PackageManager{
	{Name: "pacman", Query: []string{"pacman", "-Q"}, Install: []string{"pacman", "-S", "--needed"}, Pkgs: []string{"fprintd", "libfprint"}},
	{Name: "dnf", Query: []string{"rpm", "-q"}, Install: []string{"dnf", "install", "-y"}, Pkgs: []string{"fprintd", "fprintd-pam", "libfprint"}},
	{Name: "apt", Query: []string{"dpkg-query", "-W"}, Install: []string{"apt-get", "install", "-y"}, Pkgs: []string{"fprintd", "libpam-fprintd", "libfprint-2-2"}},
	{Name: "nix-env", Query: []string{"nix-env", "-q"}, Install: []string{"nix-env", "-iA"}, Pkgs: []string{"nixpkgs.fprintd", "nixpkgs.libfprint"}},
}

var Fingers = []string{
	"right-thumb",
	"right-index-finger",
	"right-middle-finger",
	"right-ring-finger",
	"right-little-finger",
	"left-thumb",
	"left-index-finger",
	"left-middle-finger",
	"left-ring-finger",
	"left-little-finger",
}

func DetectPackageManager() (PackageManager, bool) {
	for _, manager := range managers {
		if _, err := exec.LookPath(manager.Name); err == nil {
			return manager, true
		}
	}
	return PackageManager{}, false
}

func MissingPackages(manager PackageManager) []string {
	missing := make([]string, 0)
	for _, pkg := range manager.Pkgs {
		args := append([]string{}, manager.Query[1:]...)
		args = append(args, pkg)
		if err := exec.Command(manager.Query[0], args...).Run(); err != nil {
			missing = append(missing, pkg)
		}
	}
	return missing
}

func MissingCommands() []string {
	required := []string{"fprintd-list", "fprintd-enroll", "fprintd-verify", "fprintd-delete"}
	missing := make([]string, 0)
	for _, name := range required {
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	return missing
}

func Enrolled(user string) ([]string, error) {
	out, err := CommandOutput("fprintd-list", user)
	return ParseEnrolled(out), err
}

func ParseEnrolled(output string) []string {
	seen := make(map[string]bool)
	items := make([]string, 0)
	for _, finger := range Fingers {
		if strings.Contains(output, finger) && !seen[finger] {
			seen[finger] = true
			items = append(items, finger)
		}
	}
	return items
}

func CommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func Indent(s string) string {
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		out.WriteString("  ")
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func FilterLines(s string, needles []string) string {
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		lower := strings.ToLower(line)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				out.WriteString(line)
				out.WriteByte('\n')
				break
			}
		}
	}
	return out.String()
}
