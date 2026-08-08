package job

import (
	"atlas-data/xml"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

const jobImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="112.img">
  <imgdir name="info">
    <canvas name="icon" width="26" height="30"/>
  </imgdir>
  <imgdir name="skill">
    <imgdir name="1121000"/>
    <imgdir name="1121001"/>
    <imgdir name="1121002"/>
  </imgdir>
</imgdir>`

const emptySkillNodeXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="800.img">
  <imgdir name="skill"></imgdir>
</imgdir>`

const noSkillNodeXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="900.img">
  <imgdir name="info"></imgdir>
</imgdir>`

const mobSkillImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="MobSkill.img">
  <imgdir name="100">
    <imgdir name="level"></imgdir>
  </imgdir>
</imgdir>`

const nonNumericSkillChildXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="1.img">
  <imgdir name="skill">
    <imgdir name="1000"/>
    <imgdir name="notaskill"/>
    <imgdir name="1001"/>
  </imgdir>
</imgdir>`

// zeroJobImageXML covers job id 0 (Beginner) — document_id 0 is a legitimate
// key, and the reader must not confuse it with "no job id".
const zeroJobImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0.img">
  <imgdir name="skill">
    <imgdir name="1000"/>
  </imgdir>
</imgdir>`

func readAll(t *testing.T, data string) []RestModel {
	t.Helper()
	l, _ := test.NewNullLogger()
	ms, err := Read(l)(context.Background())(xml.FromByteArrayProvider([]byte(data)))()
	require.NoError(t, err)
	return ms
}

// writeTempImage materializes a fixture on disk for the RegisterJob tests,
// which go through xml.FromPathProvider rather than FromByteArrayProvider.
func writeTempImage(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestRead_NumericImageWithSkills_PreservesDocumentOrder(t *testing.T) {
	ms := readAll(t, jobImageXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(112), ms[0].Id)
	require.Equal(t, []uint32{1121000, 1121001, 1121002}, ms[0].Skills)
}

func TestRead_EmptySkillNode_ProducesEmptyList(t *testing.T) {
	ms := readAll(t, emptySkillNodeXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(800), ms[0].Id)
	require.NotNil(t, ms[0].Skills)
	require.Empty(t, ms[0].Skills)
}

// TestRead_MissingSkillNode_ProducesNoModel is the task-202 FR-1.1 fix. A
// numeric image with NO `skill` child is not a job document at all --
// Skill.wz/Dragon/2200.img is an Evan/Mir ANIMATION image that shares the
// real job image's name, and emitting an empty model for it let the
// document upsert (last-write-wins on (tenant, type, document_id)) blank
// the real 2200 document. Contrast TestRead_EmptySkillNode_ProducesEmptyList
// below: a PRESENT but empty `skill` node is a real job with zero skills
// (Cygnus 4th job, 1112.img) and must still produce a document. These two
// cases differ only on node presence and must never share a helper.
func TestRead_MissingSkillNode_ProducesNoModel(t *testing.T) {
	ms := readAll(t, noSkillNodeXML)
	require.Empty(t, ms)
}

func TestRead_NonNumericImage_ProducesNothingAndNoError(t *testing.T) {
	ms := readAll(t, mobSkillImageXML)
	require.Empty(t, ms)
}

func TestRead_NonNumericSkillChild_IsSkipped(t *testing.T) {
	ms := readAll(t, nonNumericSkillChildXML)
	require.Len(t, ms, 1)
	require.Equal(t, []uint32{1000, 1001}, ms[0].Skills)
}

func TestRead_JobIdZeroIsValid(t *testing.T) {
	ms := readAll(t, zeroJobImageXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(0), ms[0].Id)
	require.Equal(t, []uint32{1000}, ms[0].Skills)
}

func TestGetModelRegistry_IsSingleton(t *testing.T) {
	require.Same(t, GetModelRegistry(), GetModelRegistry())
}
