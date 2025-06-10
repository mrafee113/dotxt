package task

import (
	"dotxt/config"
	"dotxt/pkg/terrors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTodoPathArgFromCmd(t *testing.T) {
	assert := assert.New(t)
	helper := func(key, value string) *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String(key, "", "")
		cmd.SetArgs([]string{fmt.Sprintf("--%s=%s", key, value)})
		err := cmd.Execute()
		assert.Nil(err)
		return cmd
	}
	t.Run("with value", func(t *testing.T) {
		cmd := helper("testarg", "file")
		out, err := GetTodoPathArgFromCmd(cmd, "testarg")
		assert.Nil(err)
		assert.Equal("file", out)
	})
	t.Run("no value", func(t *testing.T) {
		cmd := helper("testarg", "")
		out, err := GetTodoPathArgFromCmd(cmd, "testarg")
		assert.Nil(err)
		assert.Equal(DefaultTodo, out)
	})
	t.Run("irrelavent arg", func(t *testing.T) {
		cmd := helper("testarg", "")
		out, err := GetTodoPathArgFromCmd(cmd, "irrelaventArg")
		assert.Nil(err)
		assert.Empty(out)
	})
}

func TestMkDirs(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	assert.DirExists(tmpDir)
	assert.NoDirExists(filepath.Join(tmpDir, "todos"))
	assert.NoDirExists(filepath.Join(tmpDir, "todos", "_etc"))
	assert.NoDirExists(filepath.Join(tmpDir, "todos", "_archive"))
	err = mkDirs()
	require.NoError(t, err)
	assert.DirExists(filepath.Join(tmpDir, "todos"))
	assert.DirExists(filepath.Join(tmpDir, "todos", "_etc"))
	assert.DirExists(filepath.Join(tmpDir, "todos", "_archive"))
	err = mkDirs()
	require.NoError(t, err)
}

func TestParseFilepath(t *testing.T) {
	assert := assert.New(t)
	helper := func(path string) string {
		out, err := parseFilepath(path)
		assert.Nil(err)
		return out
	}
	t.Run("empty", func(t *testing.T) {
		assert.Equal(filepath.Join(config.ConfigPath(), "todos", "todo"), helper(""))
	})
	t.Run("absolute", func(t *testing.T) {
		assert.Equal("/tmp/file", helper("/tmp/file"))
	})
	t.Run("basename", func(t *testing.T) {
		assert.Equal(filepath.Join(config.ConfigPath(), "todos", "file"), helper("file"))
	})
}

func TestPrepFileTaskFromPath(t *testing.T) {
	assert := assert.New(t)
	path := "file"
	_, err := prepFileTaskFromPath(path)
	assert.ErrorIs(err, terrors.ErrListNotInMemory)
	assert.ErrorContains(err, path)
	path, _ = parseFilepath(path)
	FileTasks[path] = make([]*Task, 0)
	_, err = prepFileTaskFromPath(path)
	assert.Nil(err)
}

func TestLocateFiles(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	mkDirs()
	CreateFile(filepath.Join(tmpDir, "todos", "file1"))
	// TODO: add support for nested files and symbolic links
	// os.MkdirAll(filepath.Join(tmpDir, "todos", "dir1"), 0755)
	// CreateFile(filepath.Join(tmpDir, "todos", "dir1", "file2"))
	FileTasks = make(map[string][]*Task)
	err = locateFiles()
	assert.Nil(err)
	_, ok := FileTasks[filepath.Join(tmpDir, "todos", "file1")]
	assert.True(ok)
	// _, ok = FileTasks[filepath.Join(tmpDir, "todos", "dir1", "file2")]
	// assert.True(ok)
}

func TestAppendToDoneFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()

	t.Run("append to non-existing", func(t *testing.T) {
		randName := filepath.Base(tmpDir)
		path := filepath.Join(config.ConfigPath(), "todos", "_etc", randName+".done")
		assert.NoFileExists(path)
		appendToDoneFile("text", randName)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("text", string(rawData))
	})
	t.Run("append to empty", func(t *testing.T) {
		name := "file"
		path := filepath.Join(config.ConfigPath(), "todos", "_etc", name+".done")
		os.WriteFile(path, []byte(""), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("text", string(rawData))
	})
	t.Run("append to file ending with \\n", func(t *testing.T) {
		name := "file2"
		path := filepath.Join(config.ConfigPath(), "todos", "_etc", name+".done")
		os.WriteFile(path, []byte("1\n2\n"), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("1\n2\ntext", string(rawData))
	})
	t.Run("append to file not ending with \\n", func(t *testing.T) {
		name := "file3"
		path := filepath.Join(config.ConfigPath(), "todos", "_etc", name+".done")
		os.WriteFile(path, []byte("1\n2"), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("1\n2\ntext", string(rawData))
	})
}

func TestRemoveFromDoneFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()
	name := "file"
	path := filepath.Join(config.ConfigPath(), "todos", "_etc", name+".done")
	os.WriteFile(path, []byte(""), 0o655)

	t.Run("validate ids", func(t *testing.T) {
		_, err := removeFromDoneFile([]int{}, "file")
		require.NotNil(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "ids is empty")
	})
	t.Run("non-existing id", func(t *testing.T) {
		_, err := removeFromDoneFile([]int{5}, name)
		require.NotNil(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "id '5' exceeds number of lines in done file")
	})
	t.Run("normal", func(t *testing.T) {
		os.WriteFile(path, []byte("a\nb\nc\n \nd\n \ne\n \nf"), 0o655)
		tasks, _ := removeFromDoneFile([]int{0}, name)
		assert.Equal("a", tasks[0])
		tasks, _ = removeFromDoneFile([]int{1, 3, 5}, name)
		assert.Equal([]string{"e", "d", "c"}, tasks)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("b\n \n \n \nf", string(rawData))
	})
}

func TestCreateFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()
	name := "file"
	path := filepath.Join(config.ConfigPath(), "todos", name)
	assert.NoFileExists(path)
	err = CreateFile(name)
	require.Nil(t, err)
	assert.FileExists(path)
	raw, err := os.ReadFile(path)
	require.Nil(t, err)
	assert.Equal("", string(raw))
	_, ok := FileTasks[path]
	assert.True(ok)
}

func TestLoadFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()
	name := "file"
	path := filepath.Join(config.ConfigPath(), "todos", name)

	t.Run("non-existing", func(t *testing.T) {
		err := LoadFile("random")
		assert.NotNil(err)
		assert.ErrorIs(err, os.ErrNotExist)
	})
	t.Run("empty file", func(t *testing.T) {
		os.WriteFile(path, []byte(""), 0o655)
		err := LoadFile(name)
		require.Nil(t, err)
		tasks, ok := FileTasks[path]
		assert.True(ok)
		assert.Empty(tasks)
	})
	t.Run("full file", func(t *testing.T) {
		os.WriteFile(path, []byte("1\n2\n3"), 0o655)
		err := LoadFile(name)
		require.Nil(t, err)
		tasks, ok := FileTasks[path]
		assert.True(ok)
		assert.Len(tasks, 3)
	})
}

func TestStoreFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()
	name := "file"
	path := filepath.Join(config.ConfigPath(), "todos", name)

	t.Run("empty tasks", func(t *testing.T) {
		os.WriteFile(path, []byte("1\n2\n"), 0o655)
		LoadFile(name)
		tasks := FileTasks[path]
		assert.Len(tasks, 2)
		FileTasks[path] = make([]*Task, 0)
		err = StoreFile(path)
		require.Nil(t, err)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("", string(raw))
	})
	t.Run("full file", func(t *testing.T) {
		os.WriteFile(path+"1", []byte("1\n2\n3\n"), 0o655)
		os.WriteFile(path, []byte(""), 0o655)
		LoadFile(name + "1")
		LoadFile(name)
		FileTasks[path] = FileTasks[path+"1"]
		err := StoreFile(path)
		require.Nil(t, err)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal(3, len(strings.Split(string(raw), "\n")))
	})
}
