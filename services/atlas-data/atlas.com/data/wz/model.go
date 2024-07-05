package wz

type FileEntry struct {
	id     string
	folder string
	name   string
	path   string
}

func (e FileEntry) Id() string {
	return e.id
}

func (e FileEntry) Folder() string {
	return e.folder
}

func (e FileEntry) Name() string {
	return e.name
}

func (e FileEntry) Path() string {
	return e.path
}

func (e FileEntry) SetId(id string) FileEntry {
	return FileEntry{id: id, folder: e.folder, name: e.name, path: e.path}
}

func (e FileEntry) SetFolder(folder string) FileEntry {
	return FileEntry{id: e.id, folder: folder, name: e.name, path: e.path}
}

func NewFileEntry(name string, path string) FileEntry {
	return FileEntry{
		name: name,
		path: path,
	}
}
