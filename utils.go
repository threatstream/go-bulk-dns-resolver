package main

import (
    "bufio"
    "io"
    "os"
)


// Originally found at http://blog.golang.org/slices.
func Append(slice []string, elements ...string) []string {
    n := len(slice)
    total := len(slice) + len(elements)
    if total > cap(slice) {
        // Reallocate. Grow to 1.5 times the new size, so we can still grow.
        newSize := total * 3 / 2 + 1
        newSlice := make([]string, total, newSize)
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[:total]
    copy(slice[n:], elements)
    return slice
}

func readLinesFromStdin(fn func(string) string) []string {
    reader := bufio.NewReader(os.Stdin)
    lines := []string{}
    for {
        line, _ /*hasMoreInLine*/, err := reader.ReadLine()
        if err == io.EOF {
            break
        } else if err != nil {
            panic(err)
        }
        lineStr := fn(string(line)) // Apply mutating callback.
        if len(lineStr) > 0 {
            lines = Append(lines, lineStr)
        }
    }
    return lines
}
