package item

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// legacyItemImgXML mirrors the serialized shape of the pre-v83 single
// String/Item.img: category children (Con/Eqp/...) with the same name/desc
// leaves the modern per-category images carry (task-172 RC-4/C-4).
const legacyItemImgXML = `<?xml version="1.0" encoding="utf-8"?>
<imgdir name="Item.img">
  <imgdir name="Con">
    <imgdir name="2000000">
      <string name="name" value="Red Potion"/>
      <string name="desc" value="Restores 50 HP."/>
    </imgdir>
  </imgdir>
  <imgdir name="Eqp">
    <imgdir name="Eqp">
      <imgdir name="Cap">
        <imgdir name="1002000">
          <string name="name" value="Blue Bandana"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="Etc">
    <imgdir name="4000000">
      <string name="name" value="Blue Snail Shell"/>
    </imgdir>
  </imgdir>
</imgdir>`

// TestInitStringFlatLegacyItemImg proves one flat pass over the legacy
// Item.img harvests flat categories AND the nested Eqp subtree — the basis
// for the C-4 adapter using InitStringFlat directly.
func TestInitStringFlatLegacyItemImg(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	path := filepath.Join(t.TempDir(), "Item.img.xml")
	require.NoError(t, os.WriteFile(path, []byte(legacyItemImgXML), 0o644))

	require.NoError(t, InitStringFlat(db)(logrus.StandardLogger())(ctx)(path))

	var rows []StringSearchIndexEntity
	require.NoError(t, db.WithContext(database.WithoutTenantFilter(ctx)).Find(&rows).Error)
	byId := map[uint32]string{}
	for _, r := range rows {
		byId[r.ItemId] = r.Name
	}
	assert.Equal(t, "Red Potion", byId[2000000], "flat Con item")
	assert.Equal(t, "Blue Bandana", byId[1002000], "nested Eqp item")
	assert.Equal(t, "Blue Snail Shell", byId[4000000], "flat Etc item")
}
