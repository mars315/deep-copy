package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/globusdigital/deep-copy/deepcopy"
	"golang.org/x/tools/go/packages"
)

func init() {
	flag.Var(&typesF, "type", "the concrete type. Multiple flags can be specified or comma-separated multiple types")
	flag.Var(&skipsF, "skip", "comma-separated field/slice/map selectors to shallow copy. Multiple flags can be specified")
	flag.Var(&outputF, "o", "the output file to write to. Defaults to deepcopy_gen.go on same dir")
}

var (
	pointerReceiverF   = flag.Bool("pointer-receiver", true, "the generated receiver type")
	maxDepthF          = flag.Int("maxdepth", 0, "max depth of deep copying")
	methodF            = flag.String("method", "DeepCopy", "deep copy method name")
	needExportPrivateF = flag.Bool("export-private", false, " deep copy private filed")
	appendOutFileF     = flag.Bool("append", false, "append to output file(not truncate file)")

	typesF  typesVal
	skipsF  skipsVal
	outputF outputVal
)

type (
	typesVal []string
	skipsVal deepcopy.SkipLists

	outputVal struct {
		file *os.File
		name string
	}
)

func (f *typesVal) String() string {
	return strings.Join(*f, ",")
}

func (f *typesVal) Set(v string) error {
	list := strings.Split(v, ",")
	*f = append(*f, list...)
	return nil
}

func (f *skipsVal) String() string {
	parts := make([]string, 0, len(*f))
	for _, m := range *f {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		parts = append(parts, strings.Join(keys, ","))
	}

	return strings.Join(parts, ",")
}

func (f *skipsVal) Set(v string) error {
	parts := strings.Split(v, ",")
	set := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		set[p] = struct{}{}
	}

	*f = append(*f, set)

	return nil
}

func (f *outputVal) String() string {
	return f.name
}

func (f *outputVal) Set(v string) error {
	if v == "-" || v == "" {
		f.name = "stdout"

		if f.file != nil {
			_ = f.file.Close()
		}
		f.file = nil

		return nil
	}

	file, err := os.OpenFile(v, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return fmt.Errorf("opening file: %v", v)
	}

	f.name = v
	f.file = file

	return nil
}

func (f *outputVal) Open(appendFile bool) (io.WriteCloser, error) {
	if f.file == nil {
		f.file = os.Stdout
	} else if !appendFile {
		err := f.file.Truncate(0)
		if err != nil {
			return nil, err
		}
	}

	return f.file, nil
}

func main() {
	flag.Parse()

	if len(typesF) == 0 || typesF[0] == "" {
		log.Fatalln("no type given")
	}

	if flag.NArg() != 1 {
		log.Fatalln("No package path given")
	}

	sl := deepcopy.SkipLists(skipsF)
	generator := deepcopy.NewGenerator(*pointerReceiverF, *needExportPrivateF, *appendOutFileF, *methodF, sl, *maxDepthF)

	if outputF.String() == "" {
		err := outputF.Set(flag.Args()[0] + "deepcopy_gen.go")
		if err != nil {
			log.Fatalln(err)
		}
	}

	output, err := outputF.Open(*appendOutFileF)
	if err != nil {
		log.Fatalln("Error initializing output file:", err)
	}

	defer output.Close()
	err = run(generator, output, flag.Args()[0], typesF)
	if err != nil {
		log.Fatalln("Error generating deep copy method:", err)
	}

}

func run(
	g deepcopy.Generator, w io.Writer, path string, types typesVal,
) error {
	packages, err := load(path)
	if err != nil {
		return fmt.Errorf("loading package: %v", err)
	}
	if len(packages) == 0 {
		return errors.New("no package found")
	}

	return g.Generate(w, types, packages[0])
}

func load(patterns string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
	}, patterns)
}
