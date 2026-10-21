// Package file is the generic File kind: a workspace folder bound to a Major File, pulled and pushed
// with `major pull`, `major push` and `major publish`. It knows nothing about what a File is used for.
package file

import (
	"fmt"
	"path"
	"strings"

	"github.com/major-technology/cli/cmd/target"
)

// Bundle is the file kind: every regular file under the root, less the skip list. priority.json is
// not part of the bundle: a File holds only renderable entries.
var Bundle = target.BundleTarget{
	Kind:    "file",
	APIPath: "files",
	Files: func(string) target.FileSet {
		return target.FileSet{
			Owns: func(rel string) bool {
				if rel == "priority.json" || path.Base(rel) == ".DS_Store" {
					return false
				}
				for _, segment := range strings.Split(path.Dir(rel), "/") {
					if target.SkippedDirs[segment] {
						return false
					}
				}
				return true
			},
			Check: func(rels []string) error {
				for _, rel := range rels {
					if strings.HasSuffix(strings.ToLower(rel), ".html") {
						return nil
					}
				}
				return fmt.Errorf("file bundle needs at least one .html entry")
			},
		}
	},
	Pack:             target.PackZip,
	ValidateByUpload: true,
}

func init() {
	target.Register("file", Bundle)
}
