package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/lipeining/godocker/cgroups"
	"github.com/lipeining/godocker/configs"
	"github.com/lipeining/godocker/container"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	exactArgs = iota
	minArgs
	maxArgs
)
const configName = "config.json"

func checkArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name
	switch checkType {
	case exactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case minArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case maxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}

	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

func logrusToStderr() bool {
	l, ok := logrus.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

func pathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func revisePidFile(context *cli.Context) error {
	pidFile := context.String("pid-file")
	if pidFile == "" {
		return nil
	}

	// convert pid-file to an absolute path so we can write to the right
	// file after chdir to bundle
	pidFile, err := filepath.Abs(pidFile)
	if err != nil {
		return err
	}
	return context.Set("pid-file", pidFile)
}

// parseBoolOrAuto returns (nil, nil) if s is empty or "auto"
func parseBoolOrAuto(s string) (*bool, error) {
	if s == "" || strings.ToLower(s) == "auto" {
		return nil, nil
	}
	b, err := strconv.ParseBool(s)
	return &b, err
}

// WriteJSON writes the provided struct v to w using standard json marshaling
func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// CleanPath makes a path safe for use with filepath.Join. This is done by not
// only cleaning the path, but also (if the path is relative) adding a leading
// '/' and cleaning it (then removing the leading '/'). This ensures that a
// path resulting from prepending another path will always resolve to lexically
// be a subdirectory of the prefixed path. This is all done lexically, so paths
// that include symlinks won't be safe as a result of using CleanPath.
func CleanPath(path string) string {
	// Deal with empty strings nicely.
	if path == "" {
		return ""
	}

	// Ensure that all paths are cleaned (especially problematic ones like
	// "/../../../../../" which can cause lots of issues).
	path = filepath.Clean(path)

	// If the path isn't absolute, we need to do more processing to fix paths
	// such as "../../../../<etc>/some/path". We also shouldn't convert absolute
	// paths to relative ones.
	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		// This can't fail, as (by definition) all paths are relative to root.
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}

	// Clean the path again for good measure.
	return filepath.Clean(path)
}

var errEmptyID = errors.New("container id is empty")

func resolveBundle(context *cli.Context) error {
	bundle := context.String("bundle")
	if bundle != "" {
		if err := os.Chdir(bundle); err != nil {
			return nil
		}
	}
	return nil
}

func loadConfig() (*configs.Config, error) {
	exist, err := pathExist(configName)
	if err != nil {
		return nil, err
	}
	if exist == false {
		return nil, fmt.Errorf("path is hold no config.json")
	}
	f, err := os.Open(configName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config *configs.Config
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	return config, validateConfig(config)
}

func validateConfig(config *configs.Config) error {
	if config == nil {
		return errors.New("config property must not be empty")
	}
	if config.Rootfs == "" {
		return errors.New("Rootfs property must not be empty")
	}
	return nil
}

func GenContainerID(n int) string {
	letterBytes := "0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func ensureRoot(root, id string) error {
	// The --root parameter tells runc where to store the container state.
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	containerRoot, err := securejoin.SecureJoin(abs, id)
	if err != nil {
		return err
	}
	exist, err := pathExist(containerRoot)
	if err != nil {
		return err
	} else if exist {
		return nil
	}
	if err := os.MkdirAll(containerRoot, 0711); err != nil {
		return err
	}
	// if err := os.Chown(containerRoot, unix.Geteuid(), unix.Getegid()); err != nil {
	// 	return err
	// }
	return nil
}

func newProcess(context *cli.Context) *container.Process {
	args := make([]string, 0)
	for _, arg := range context.Args().Tail() {
		args = append(args, arg)
	}
	envs := context.StringSlice("e")
	// 使用 runtime-spec spec.Specs 中的 process 对象
	// 可以设置 pwd, args, env, cmd 等属性
	cwd := context.String("cwd")
	if cwd == "" {
		cwd = "/"
	}
	process := &container.Process{
		Cwd:  cwd,
		Args: args,
		Env:  envs,
	}
	return process
}

func createContainer(context *cli.Context) (*container.Container, error) {
	id := GenContainerID(10)
	context.Set("id", id)
	root := context.GlobalString("root")
	if err := ensureRoot(root, id); err != nil {
		return nil, err
	}
	if err := resolveBundle(context); err != nil {
		return nil, err
	}
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}
	cgroupManager, err := cgroups.NewManager(id, config)
	if err != nil {
		return nil, err
	}
	return container.NewContainer(context, config, cgroupManager)
}

func startContainer(context *cli.Context) (int, error) {
	c, err := createContainer(context)
	if err != nil {
		return -1, err
	}
	process := newProcess(context)
	initProcess, err := container.NewInitProcess(process, c)
	if err != nil {
		return -1, err
	}
	fmt.Println(c, initProcess)
	// 创建对应的 roofs system
	imageName := "busybox"
	if err := container.NewWorkSpace(c, imageName); err != nil {
		return -1, err
	}
	if err := c.StartInit(initProcess); err != nil {
		return -1, err
	}
	return 0, nil
}

func getContainer(context *cli.Context, id string) (*container.Container, error) {
	if id == "" {
		return nil, errEmptyID
	}
	root := context.GlobalString("root")
	if root == "" {
		return nil, fmt.Errorf("invalid root")
	}
	containerRoot, err := securejoin.SecureJoin(root, id)
	if err != nil {
		return nil, err
	}
	state, err := loadState(containerRoot, id)
	if err != nil {
		return nil, err
	}
	c := &container.Container{
		InitProcess: container.InitProcess{},
		Id:          id,
		Config:      &state.Config,
		InitPath:    "/proc/self/exe",
		InitArgs:    []string{"init"},
		Root:        containerRoot,
		Created:     state.Created,
	}
	if err := c.RefreshState(); err != nil {
		return nil, err
	}
	return c, nil
}

func loadState(root, id string) (*container.State, error) {
	stateFilePath, err := securejoin.SecureJoin(root, container.StateFilename)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("container %q does not exist", id)
		}
		return nil, err
	}
	defer f.Close()
	var state *container.State
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}
	return state, nil
}

var idRegex = regexp.MustCompile(`^[\w+-\.]+$`)

func validateID(id string) error {
	if !idRegex.MatchString(id) {
		return fmt.Errorf("invalid id format: %v", id)
	}

	return nil
}
