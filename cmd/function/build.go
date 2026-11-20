package function

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/major-technology/cli/cmd/target"
)

// bundlePath is where the built module sits in a pushed archive. It is never
// written to the workspace.
const bundlePath = buildDir + "/bundle.js"

// buildEntry re-exports the function as the module's default export and its
// function.json as `config`, the two exports the function runner reads.
const buildEntry = "export { default } from \"./index\";\nimport config from \"./" + configFile + "\";\nexport { config };\n"

// buildBundle bundles index.ts, with the npm dependencies and clients in the
// function's own folder inlined, into the single minified ES module the
// function runner loads.
func buildBundle(root string) (map[string][]byte, error) {
	result := api.Build(api.BuildOptions{
		Stdin:             &api.StdinOptions{Contents: buildEntry, ResolveDir: root, Loader: api.LoaderTS},
		AbsWorkingDir:     root,
		Bundle:            true,
		Format:            api.FormatESModule,
		Platform:          api.PlatformNode,
		Target:            api.ES2022,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Write:             false,
		LogLevel:          api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		messages := make([]string, 0, len(result.Errors))
		for _, message := range result.Errors {
			if message.Location != nil {
				messages = append(messages, fmt.Sprintf("%s:%d: %s", message.Location.File, message.Location.Line, message.Text))
			} else {
				messages = append(messages, message.Text)
			}
		}
		return nil, &target.InvalidBundleError{Err: fmt.Errorf("function failed to build: %s", strings.Join(messages, "; "))}
	}
	if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("function build produced %d files, expected 1", len(result.OutputFiles))
	}
	return map[string][]byte{bundlePath: result.OutputFiles[0].Contents}, nil
}
