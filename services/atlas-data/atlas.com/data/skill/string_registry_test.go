package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func writeTempFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "skill.img.xml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestInitString_PopulatesDesc(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	const fixture = `<imgdir name="root">
  <imgdir name="1001">
    <string name="name" value="Recovery"/>
    <string name="desc" value="Recovers HP over time."/>
  </imgdir>
</imgdir>`

	path := writeTempFixture(t, fixture)

	require.NoError(t, InitString(tn, path))
	got, err := GetSkillStringRegistry().Get(tn, "1001")
	require.NoError(t, err)
	require.Equal(t, "Recovery", got.Name())
	require.Equal(t, "Recovers HP over time.", got.Desc())
}
