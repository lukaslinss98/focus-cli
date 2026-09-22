package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/lukas/focus/internal/config"
	"github.com/lukas/focus/internal/domain"
	"github.com/lukas/focus/internal/hosts"
)

type Application struct {
	out io.Writer
}

func New(out io.Writer) *Application {
	return &Application{out: out}
}

func (a *Application) Run(arguments []string) error {
	if len(arguments) == 0 {
		a.usage()
		return nil
	}
	if arguments[0] == "__sync" {
		return a.sync(arguments[1:])
	}

	store, err := defaultStore()
	if err != nil {
		return err
	}

	switch arguments[0] {
	case "add":
		return a.add(store, arguments[1:])
	case "remove":
		return a.remove(store, arguments[1:])
	case "on":
		return a.setEnabled(store, true, arguments[1:])
	case "off":
		return a.setEnabled(store, false, arguments[1:])
	case "list":
		return a.list(store, arguments[1:])
	case "status":
		return a.status(store, arguments[1:])
	case "help", "--help", "-h":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func (a *Application) add(store *config.Store, arguments []string) error {
	if len(arguments) != 1 {
		return errors.New("usage: focus add <website>")
	}
	name, err := domain.Normalize(arguments[0])
	if err != nil {
		return err
	}
	current, err := store.Load()
	if err != nil {
		return err
	}
	if contains(current.Domains, name) {
		fmt.Fprintf(a.out, "%s is already managed\n", name)
		return nil
	}
	current.Domains = append(current.Domains, name)
	if err := store.Save(current); err != nil {
		return err
	}
	if current.Enabled {
		if err := syncPrivileged(store.Path()); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.out, "Added %s and www.%s\n", name, name)
	return nil
}

func (a *Application) remove(store *config.Store, arguments []string) error {
	if len(arguments) != 1 {
		return errors.New("usage: focus remove <website>")
	}
	name, err := domain.Normalize(arguments[0])
	if err != nil {
		return err
	}
	current, err := store.Load()
	if err != nil {
		return err
	}
	if !contains(current.Domains, name) {
		return fmt.Errorf("%s is not managed", name)
	}
	current.Domains = without(current.Domains, name)
	if err := store.Save(current); err != nil {
		return err
	}
	if current.Enabled {
		if err := syncPrivileged(store.Path()); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.out, "Removed %s and www.%s\n", name, name)
	return nil
}

func (a *Application) setEnabled(store *config.Store, enabled bool, arguments []string) error {
	if len(arguments) != 0 {
		return fmt.Errorf("usage: focus %s", map[bool]string{true: "on", false: "off"}[enabled])
	}
	current, err := store.Load()
	if err != nil {
		return err
	}
	wasEnabled := current.Enabled
	current.Enabled = enabled
	if err := store.Save(current); err != nil {
		return err
	}
	if err := syncPrivileged(store.Path()); err != nil {
		return err
	}
	if wasEnabled == enabled {
		fmt.Fprintf(a.out, "Focus is already %s\n", map[bool]string{true: "on", false: "off"}[enabled])
		return nil
	}
	fmt.Fprintf(a.out, "Focus is %s\n", map[bool]string{true: "on", false: "off"}[enabled])
	return nil
}

func (a *Application) list(store *config.Store, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: focus list")
	}
	current, err := store.Load()
	if err != nil {
		return err
	}
	if len(current.Domains) == 0 {
		fmt.Fprintln(a.out, "No websites are managed.")
		return nil
	}
	for _, name := range current.Domains {
		fmt.Fprintf(a.out, "%s\twww.%s\n", name, name)
	}
	return nil
}

func (a *Application) status(store *config.Store, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: focus status")
	}
	current, err := store.Load()
	if err != nil {
		return err
	}
	installed, err := hosts.SystemManager().HasManagedBlock()
	if err != nil {
		return err
	}
	state := "off"
	if current.Enabled {
		state = "on"
	}
	fmt.Fprintf(a.out, "Focus: %s\nManaged websites: %d\nHosts block installed: %t\nConfig: %s\n", state, len(current.Domains), installed, store.Path())
	return nil
}

func (a *Application) sync(arguments []string) error {
	if len(arguments) != 1 {
		return errors.New("internal sync requires a configuration path")
	}
	store := config.NewStore(arguments[0])
	current, err := store.Load()
	if err != nil {
		return err
	}
	changed, err := hosts.SystemManager().Sync(current.Enabled, current.Domains)
	if err != nil {
		return err
	}
	if changed {
		if err := flushDNSCache(); err != nil {
			return err
		}
	}
	return nil
}

func syncPrivileged(configPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find focus executable: %w", err)
	}
	command := exec.Command("sudo", executable, "__sync", configPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("synchronize hosts file: %w", err)
	}
	return nil
}

func flushDNSCache() error {
	for _, command := range [][]string{{"dscacheutil", "-flushcache"}, {"killall", "-HUP", "mDNSResponder"}} {
		if err := exec.Command(command[0], command[1:]...).Run(); err != nil {
			return fmt.Errorf("flush DNS cache: %w", err)
		}
	}
	return nil
}

func defaultStore() (*config.Store, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	return config.NewStore(path), nil
}

func contains(domains []string, name string) bool {
	for _, value := range domains {
		if value == name {
			return true
		}
	}
	return false
}

func without(domains []string, name string) []string {
	result := make([]string, 0, len(domains)-1)
	for _, value := range domains {
		if value != name {
			result = append(result, value)
		}
	}
	return result
}

func (a *Application) usage() {
	fmt.Fprint(a.out, `Usage: focus <command>

Commands:
  add <website>     Add a website and its www variant
  remove <website>  Remove a managed website
  on                Enable blocking
  off               Disable blocking
  list              List managed websites
  status            Show current state
`)
}
