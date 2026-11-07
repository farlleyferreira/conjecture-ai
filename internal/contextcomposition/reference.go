package contextcomposition

import "database/sql"

// Reference is a composition-by-reference pointer to another
// context's content - "Four composition modes": "Composition by
// reference does not copy content": "WHEN a context is composed by
// reference THEN the composed context points to the source context's
// content rather than duplicating it." A Reference holds only Name
// and (optionally) Seq - never a copy of the source's own content -
// so there is nothing stored here that can ever go stale against it.
type Reference struct {
	// Name is the source context's name.
	Name string
	// Seq is the specific version pointed to, or 0 to always follow
	// whatever the source's latest version currently is - the same
	// zero-means-latest convention [VersionRef] already uses.
	Seq int
}

// ResolveReference reads the content ref currently points to, live
// from db every time - via [LatestVersion] when Seq is 0, [VersionAt]
// otherwise. This is what makes composition-by-reference not copy
// content: resolving the same Reference again after the source
// gained a new version returns that new version's content, not
// whatever was current when the Reference was first created.
func ResolveReference(db *sql.DB, ref Reference) (Version, error) {
	if ref.Seq == 0 {
		return LatestVersion(db, ref.Name)
	}
	return VersionAt(db, ref.Name, ref.Seq)
}
