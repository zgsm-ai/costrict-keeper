package utils

type VersionOverview struct {
	VersionId   VersionNumber `json:"versionId"`      //版本号，采用SemVer标准
	PackageType PackageType   `json:"packageType"`    //包类型: exec/conf
	FileName    string        `json:"fileName"`       //被打包的文件的名字
	Size        uint64        `json:"size,omitempty"` //包文件大小
	Build       string        `json:"build"`          //构建信息：Tag/Branch信息 CommitID BuildTime
	Description string        `json:"description"`    //版本描述，含有更丰富的可读信息
}

type PlatformOverview struct {
	Os       string            `json:"os"`
	Arch     string            `json:"arch"`
	Versions []VersionOverview `json:"versions"`
}

type PackageOverview struct {
	PackageName string             `json:"packageName"`
	Platforms   []PlatformOverview `json:"platforms"`
}

/**
 *	云端包信息概览
 */
type PackagesOverview struct {
	Packages map[string]PackageOverview `json:"packages"`
}
