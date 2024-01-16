package main

import (
    "fmt"
    "github.com/antithesishq/antithesis-sdk-go/assert"
    "os"
    "path"
    "strings"
)

func main() {
    fmt.Printf(assert.Version())


    fmt.Printf("\n")
    check_var := "LD_LIBRARY_PATH"
    args := os.Args
    if len(args) > 1 {
        check_var = args[1]
    }
    path_list := os.Getenv(check_var)
    lib_paths := AllPaths(path_list)
    for _, p := range lib_paths {
        fmt.Printf("Path => %q\n", p)
        // if fullname, exists := ExistsAtPath("libvoidstar.so", p); exists {
        //     fmt.Printf("Found %q\n", fullname)
        //     return fullname
        // }
    }
    filename := "libvoidstar.so"
    if fullpath := FullPathForFile(lib_paths, filename ); fullpath != "" {
        fmt.Printf("Found %q\n", fullpath)
        return
    }
    fmt.Printf("Unable to find %q\n",  filename)
}

func FullPathForFile(possible_paths []string, filename string) string {
    for _, p := range possible_paths {
        fmt.Printf("Path => %q\n", p)
        if fullname, exists := ExistsAtPath("libvoidstar.so", p); exists {
            fmt.Printf("Found %q\n", fullname)
            return fullname
        }
    }
    return ""
}

func AllPaths(path_list string) []string {
    all_paths := []string{}
    s := strings.TrimSpace(path_list)
    if len(s) == 0 {
        return all_paths
    }
    parts := strings.Split(path_list, ":")
    for _, part := range parts {
        all_paths = append(all_paths, strings.TrimSpace(part))
    }
    return all_paths
}

func ExistsAtPath(filename string, filepath string) (fullname string, exists bool) {
    fullname = path.Join(filepath, filename)
    exists = false
    if _, err := os.Stat(path); err == nil {
        exists = true
        return
    }
    return
}
