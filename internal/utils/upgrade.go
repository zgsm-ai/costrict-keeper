package utils

import (
	"bufio"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

/**
 *	包类型枚举
 */
type PackageType string

const (
	PackageTypeExec PackageType = "exec"
	PackageTypeConf PackageType = "conf"
)

/**
 *	版本编号
 */
type VersionNumber struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Micro int `json:"micro"`
}

/**
 *	包版本的描述&签名信息，用于验证包的正确性
 */
type PackageVersion struct {
	PackageName  string        `json:"packageName"`    //包名字
	PackageType  PackageType   `json:"packageType"`    //包类型: exec/conf
	FileName     string        `json:"fileName"`       //被打包的文件的名字
	Os           string        `json:"os"`             //操作系统名:linux/windows
	Arch         string        `json:"arch"`           //硬件架构
	Size         uint64        `json:"size,omitempty"` //包文件大小
	Checksum     string        `json:"checksum"`       //Md5散列值
	Sign         string        `json:"sign"`           //签名，使用私钥签的名，需要用对应公钥验证
	ChecksumAlgo string        `json:"checksumAlgo"`   //固定为“md5”
	VersionId    VersionNumber `json:"versionId"`      //版本号，采用SemVer标准
	Build        string        `json:"build"`          //构建信息：Tag/Branch信息 CommitID BuildTime
	Description  string        `json:"description"`    //版本描述，含有更丰富的可读信息
}

/**
 *	一个package版本的地址信息
 */
type VersionAddr struct {
	VersionId VersionNumber `json:"versionId"` //版本的地址信息
	AppUrl    string        `json:"appUrl"`    //包地址
	InfoUrl   string        `json:"infoUrl"`   //包描述信息(PackageVersion)文件的地址
}

/**
 *	指定平台的关键信息，比如，最新版本，版本列表（描述一个硬件平台/操作系统对应的包列表）
 */
type PlatformInfo struct {
	PackageName string        `json:"packageName"`
	Os          string        `json:"os"`
	Arch        string        `json:"arch"`
	Newest      VersionAddr   `json:"newest"`
	Versions    []VersionAddr `json:"versions"`
}

/**
 *	平台标识
 */
type PlatformId struct {
	Os   string `json:"os"`
	Arch string `json:"arch"`
}

/**
 *	包目录（软件包的系统，平台，版本目录）
 */
type PackageDirectory struct {
	PackageName string       `json:"packageName"`
	Platforms   []PlatformId `json:"platforms"`
}

/**
 *	云端可供下载的包列表
 */
type PackageList struct {
	Packages []string `json:"packages"`
}

type UpgradeConfig struct {
	PublicKey   string //用来验证包签名的公钥
	BaseUrl     string //保存安装包的服务器的基地址
	BaseDir     string //costrict数据所在的基路径
	InstallDir  string //软件包的安装路径
	PackageDir  string //保存下载软件包的数据文件&包描述文件
	PackageName string //包名称
	TargetPath  string //指定安装目标路径(及文件名)
	Os          string //操作系统名
	Arch        string //硬件平台名
	NoSetPath   bool   //不需要设置PATH。设置PATH可以让程序所在路径被自动搜索
	CleanCache  bool   //清理掉该版本所有包文件的缓存
}

// const SHENMA_PUBLIC_KEY = `-----BEGIN PUBLIC KEY-----
// MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwClPrRPGCOXcWPFMPIPc
// Hn5angPRwuIvwSGle/O7VaZfaTuplMVa2wUPzWv1AfmKpENMm0pf0uhnTyfH3gnR
// C46rNeMmBcLg8Jd7wTWXtik0IN7CREOQ6obIiMY4Sbx25EPHPf8SeqvPpFq8uOEM
// YqRUQbPaY5+mgkDZMy68hJDUUstapBQovjSlnLXjG2pULWKIJF2g0gGWvS4LGznP
// Uvrq2U1QVpsja3EtoLq8jF3UcLJWVZt2pMd5H9m3ULBKFzpu7ix+wb3ebRr6JtUI
// bMzLAZ0BM0wxlpDmp1GYVag+Ll3w2o3LXLEB08soABD0wdD03Sp7flkbebgAxd1b
// vwIDAQAB
// -----END PUBLIC KEY-----`

const SHENMA_PUBLIC_KEY = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp/yvHEtGy09fNgZO2a/e
oyjEvBqVEjNf9RRf8r5QLeXI/InJGS323faqrVAtEjbOhq1R0KuAYISyFRzPvJYa
aBdlaDpXOY0UJxz6C/hLSAl2ohn/SvCycYVucrjnPUAwCqDNaLLjyqyTdsSXNh3d
QHgyBM16LD8oqFHj+/dxlMNxv+FIcc6WeN9F7BmTmvbHt5jBqBxBhXtlR8lx7F/H
AIMDOcw+6STgS2RFFnTRrBl8ZgJPBUavczm0TY4a9gUErfTnb8zBHtH6K4OPsvEF
Nimo+oDprwaVnIIPm1UvZtc/Qe/6OD0emoVovSzRYhbaqVPWgKqPNiitW9JZvuV3
nwIDAQAB
-----END PUBLIC KEY-----`

const SHENMA_BASE_URL = "https://zgsm.sangfor.com/costrict"

func (cfg *UpgradeConfig) Correct() {
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}
	if cfg.Os == "" {
		cfg.Os = runtime.GOOS
	}
	if cfg.BaseDir == "" {
		cfg.BaseDir = getCostrictDir()
	}
	if cfg.InstallDir == "" {
		cfg.InstallDir = filepath.Join(cfg.BaseDir, "bin")
	}
	if cfg.PackageDir == "" {
		cfg.PackageDir = filepath.Join(cfg.BaseDir, "package")
	}
	if cfg.BaseUrl == "" {
		cfg.BaseUrl = SHENMA_BASE_URL
	}
	if cfg.PublicKey == "" {
		cfg.PublicKey = SHENMA_PUBLIC_KEY
	}
}

/**
 *	从云端获取一个文件的内容
 */
func GetBytes(urlStr string, params map[string]string) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("GetBytes: %v", err)
	}
	vals := make(url.Values)
	for k, v := range params {
		vals.Set(k, v)
	}
	req.URL.RawQuery = vals.Encode()

	rsp, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("GetBytes: %v", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		rspBody, _ := io.ReadAll(rsp.Body)
		return rspBody, fmt.Errorf("GetBytes('%s?%s') code:%d, error:%s",
			urlStr, req.URL.RawQuery, rsp.StatusCode, string(rspBody))
	}
	return io.ReadAll(rsp.Body)
}

/**
 *	从服务器获取一个文件
 */
func GetFile(urlStr string, params map[string]string, savePath string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("GetFile('%s') failed: %v", urlStr, err)
	}
	vals := make(url.Values)
	for k, v := range params {
		vals.Set(k, v)
	}
	req.URL.RawQuery = vals.Encode()

	rsp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GetFile('%s') failed: %v", urlStr, err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		rspBody, _ := io.ReadAll(rsp.Body)
		return fmt.Errorf("GetFile('%s', '%s') code: %d, error:%s",
			urlStr, req.URL.RawQuery, rsp.StatusCode, string(rspBody))
	}

	// 创建一个文件用于保存
	if err = os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("GetFile('%s'): MkdirAll('%s') error:%v", urlStr, savePath, err)
	}
	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("GetFile('%s'): create('%s') error: %v", urlStr, savePath, err)
	}
	defer out.Close()

	// 然后将响应流和文件流对接起来
	_, err = io.Copy(out, rsp.Body)
	if err != nil {
		return fmt.Errorf("GetFile('%s'): copy error: %v", urlStr, err)
	}
	return err
}

/**
 *	解析版本字符串，得到版本号
 */
func ParseVersion(verstr string) (VersionNumber, error) {
	vers := strings.Split(verstr, ".")
	id := VersionNumber{}
	if len(vers) != 3 {
		return id, fmt.Errorf("invalid version string")
	}
	var err error
	id.Major, err = strconv.Atoi(vers[0])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	id.Minor, err = strconv.Atoi(vers[1])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	id.Micro, err = strconv.Atoi(vers[2])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	return id, nil
}

/**
 *	打印版本号
 */
func PrintVersion(ver VersionNumber) string {
	return fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Micro)
}

/**
 *	获取本地已安装包的版本
 */
func GetLocalVersion(cfg UpgradeConfig) (VersionNumber, error) {
	packageFileName := filepath.Join(cfg.PackageDir, fmt.Sprintf("%s.json", cfg.PackageName))
	var pkg PackageVersion
	bytes, err := os.ReadFile(packageFileName)
	if err != nil {
		return VersionNumber{}, err
	}
	if err := json.Unmarshal(bytes, &pkg); err != nil {
		return VersionNumber{}, err
	}
	return pkg.VersionId, nil
}

/**
 *	从远程库获取包版本
 */
func GetRemoteVersions(cfg UpgradeConfig) (PlatformInfo, error) {
	//	<base-url>/<package>/<os>/<arch>/platform.json
	urlStr := fmt.Sprintf("%s/%s/%s/%s/platform.json",
		cfg.BaseUrl, cfg.PackageName, cfg.Os, cfg.Arch)

	bytes, err := GetBytes(urlStr, nil)
	if err != nil {
		return PlatformInfo{}, err
	}
	vers := &PlatformInfo{}
	if err = json.Unmarshal(bytes, vers); err != nil {
		return *vers, fmt.Errorf("GetRemoteVersions('%s') unmarshal error: %v", urlStr, err)
	}
	return *vers, nil
}

func GetRemotePlatforms(cfg UpgradeConfig) (PackageDirectory, error) {
	//	<base-url>/<package>/platforms.json
	urlStr := fmt.Sprintf("%s/%s/platforms.json",
		cfg.BaseUrl, cfg.PackageName)

	bytes, err := GetBytes(urlStr, nil)
	if err != nil {
		return PackageDirectory{}, err
	}
	plats := &PackageDirectory{}
	if err = json.Unmarshal(bytes, plats); err != nil {
		return *plats, fmt.Errorf("GetRemotePlatforms('%s') unmarshal error: %v", urlStr, err)
	}
	return *plats, nil
}

func GetRemotePackages(cfg UpgradeConfig) (PackageList, error) {
	//	<base-url>/packages.json
	urlStr := fmt.Sprintf("%s/packages.json", cfg.BaseUrl)

	bytes, err := GetBytes(urlStr, nil)
	if err != nil {
		return PackageList{}, err
	}
	pkgs := &PackageList{}
	if err = json.Unmarshal(bytes, pkgs); err != nil {
		return *pkgs, fmt.Errorf("GetRemotePackages('%s') unmarshal error: %v", urlStr, err)
	}
	return *pkgs, nil
}

func GetRemoteOverview(cfg UpgradeConfig) (PackagesOverview, error) {
	//	<base-url>/packages-overview.json
	urlStr := fmt.Sprintf("%s/packages-overview.json", cfg.BaseUrl)

	bytes, err := GetBytes(urlStr, nil)
	if err != nil {
		return PackagesOverview{}, err
	}
	pkgs := PackagesOverview{}
	if err = json.Unmarshal(bytes, &pkgs); err != nil {
		return pkgs, fmt.Errorf("GetRemoteOverview('%s') unmarshal error: %v", urlStr, err)
	}
	return pkgs, nil
}

/**
 *	比较版本
 */
func CompareVersion(local, remote VersionNumber) int {
	if local.Major != remote.Major {
		return local.Major - remote.Major
	}
	if local.Minor != remote.Minor {
		return local.Minor - remote.Minor
	}
	return local.Micro - remote.Micro
}

/**
 *	获取costrict目录结构设定
 */
func getCostrictDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".costrict")
}

/**
 *	获取包(需要校验保证包的合法性)
 */
func GetPackage(cfg UpgradeConfig, specVer *VersionNumber) (PackageVersion, bool, error) {
	var pkg PackageVersion
	var curVer VersionNumber

	//	获取本地版本信息
	packageFileName := filepath.Join(cfg.PackageDir, fmt.Sprintf("%s.json", cfg.PackageName))
	bytes, err := os.ReadFile(packageFileName)
	if err == nil {
		if err := json.Unmarshal(bytes, &pkg); err == nil {
			curVer = pkg.VersionId
		}
	}
	//	获取云端的最新版本
	vers, err := GetRemoteVersions(cfg)
	if err != nil {
		log.Printf("Get remote versions for package '%s' failed: %v\n", cfg.PackageName, err)
		return pkg, false, err
	}

	addr := VersionAddr{}
	if specVer != nil { //升级指定版本
		//	检查指定版本specVer在不在版本列表中
		found := false
		for _, v := range vers.Versions {
			if CompareVersion(v.VersionId, *specVer) == 0 {
				addr = v
				found = true
				break
			}
		}
		if !found {
			log.Printf("Specified version %s not found for package '%s'\n", PrintVersion(*specVer), cfg.PackageName)
			return pkg, false, fmt.Errorf("version %s isn't exist", PrintVersion(*specVer))
		}
	} else { //升级最新版本
		//	比较当前最新版本，看是否有必要升级
		ret := CompareVersion(curVer, vers.Newest.VersionId)
		if ret >= 0 {
			return pkg, false, nil
		}
		addr = vers.Newest
	}
	//	获取云端升级包的描述信息
	data, err := GetBytes(cfg.BaseUrl+addr.InfoUrl, nil)
	if err != nil {
		log.Printf("Get package info from '%s' failed: %v\n", addr.InfoUrl, err)
		return pkg, false, err
	}
	if err = json.Unmarshal(data, &pkg); err != nil {
		log.Printf("Unmarshal package info from '%s' failed: %v\n", addr.InfoUrl, err)
		return pkg, false, fmt.Errorf("unmarshal '%s' error: %v", addr.InfoUrl, err)
	}
	if pkg.FileName == "" {
		pkg.FileName = pkg.PackageName
	}
	cacheDir := filepath.Join(cfg.PackageDir, PrintVersion(addr.VersionId))
	if err = os.MkdirAll(cfg.InstallDir, 0775); err != nil {
		log.Printf("Create install directory '%s' failed: %v\n", cfg.InstallDir, err)
		return pkg, false, fmt.Errorf("MkdirAll('%s') error: %v", cfg.InstallDir, err)
	}
	if err = os.MkdirAll(cacheDir, 0775); err != nil {
		log.Printf("Create cache directory '%s' failed: %v\n", cacheDir, err)
		return pkg, false, fmt.Errorf("MkdirAll('%s') error: %v", cacheDir, err)
	}
	//	下载包
	dataFname := filepath.Join(cacheDir, pkg.FileName)
	if err = GetFile(cfg.BaseUrl+addr.AppUrl, nil, dataFname); err != nil {
		log.Printf("Download package from '%s' to '%s' failed: %v\n", addr.AppUrl, dataFname, err)
		return pkg, false, err
	}
	//	检查下载包的MD5
	_, md5str, err := CalcFileMd5(dataFname)
	if err != nil {
		log.Printf("Calculate MD5 for file '%s' failed: %v\n", dataFname, err)
		return pkg, false, err
	}
	if md5str != pkg.Checksum {
		log.Printf("MD5 checksum mismatch for package '%s'. Expected: %s, Actual: %s\n", addr.AppUrl, pkg.Checksum, md5str)
		return pkg, false, fmt.Errorf("checksum error: %s", addr.AppUrl)
	}
	//	检查签名，防止包被篡改
	sig, err := hex.DecodeString(pkg.Sign)
	if err != nil {
		log.Printf("Decode signature for package '%s' failed: %v\n", pkg.PackageName, err)
		return pkg, false, fmt.Errorf("decode sign error: %v", err)
	}
	if err = VerifySign([]byte(cfg.PublicKey), sig, []byte(md5str)); err != nil {
		log.Printf("Verify signature for package '%s' failed: %v\n", pkg.PackageName, err)
		return pkg, false, fmt.Errorf("verify sign error: %v", err)
	}
	//	把包描述文件保存到包文件目录
	packageFileName = filepath.Join(cfg.PackageDir, fmt.Sprintf("%s-%s.json", cfg.PackageName, PrintVersion(pkg.VersionId)))
	if err := os.WriteFile(packageFileName, data, 0644); err != nil {
		log.Printf("Write package info file '%s' failed: %v\n", packageFileName, err)
		return pkg, false, err
	}
	return pkg, true, nil
}

/**
 *	激活版本ver的包，令其成为当前版本
 */
func ActivatePackage(cfg UpgradeConfig, ver VersionNumber) error {
	var pkg PackageVersion

	packageFileName := filepath.Join(cfg.PackageDir, fmt.Sprintf("%s-%s.json", cfg.PackageName, PrintVersion(ver)))
	data, err := os.ReadFile(packageFileName)
	if err != nil {
		log.Printf("Read package file '%s' failed: %v\n", packageFileName, err)
		return err
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		log.Printf("Unmarshal package data from '%s' failed: %v\n", packageFileName, err)
		return err
	}
	cacheDir := filepath.Join(cfg.PackageDir, PrintVersion(ver))
	dataFname := filepath.Join(cacheDir, pkg.FileName)
	//	把下载的包安装到正式目录
	if err = installPackage(cfg, pkg, dataFname); err != nil {
		log.Printf("Install package '%s' failed: %v\n", dataFname, err)
		return fmt.Errorf("installPackage('%s') error: %v", dataFname, err)
	}
	packageFileName = filepath.Join(cfg.PackageDir, fmt.Sprintf("%s.json", cfg.PackageName))
	if err := os.WriteFile(packageFileName, data, 0644); err != nil {
		log.Printf("Write current package file '%s' failed: %v\n", packageFileName, err)
		return err
	}
	if cfg.CleanCache {
		if err := os.RemoveAll(cacheDir); err != nil {
			log.Printf("Remove cache directory '%s' failed: %v\n", cacheDir, err)
		}
		packageFileName = filepath.Join(cfg.PackageDir, fmt.Sprintf("%s-%s.json", cfg.PackageName, PrintVersion(pkg.VersionId)))
		if err := os.Remove(packageFileName); err != nil {
			log.Printf("Remove version package file '%s' failed: %v\n", packageFileName, err)
		}
	}
	return nil
}

/**
 *	升级包
 */
func UpgradePackage(cfg UpgradeConfig, specVer *VersionNumber) (VersionNumber, bool, error) {
	pkg, upgraded, err := GetPackage(cfg, specVer)
	if err != nil {
		return VersionNumber{}, false, err
	}
	if !upgraded { //不需要更新，所以不需要激活
		return pkg.VersionId, false, nil
	}
	if err := ActivatePackage(cfg, pkg.VersionId); err != nil {
		return pkg.VersionId, false, err
	}
	return pkg.VersionId, true, nil
}

/**
 *	保存包数据文件
 */
func savePackageData(cfg UpgradeConfig, pkg PackageVersion, tmpFname string) error {
	var targetFileName string
	if cfg.TargetPath != "" {
		targetFileName = cfg.TargetPath
	} else {
		targetFileName = filepath.Join(cfg.InstallDir, pkg.FileName)
	}
	if err := os.MkdirAll(filepath.Dir(targetFileName), 0755); err != nil {
		return err
	}
	if err := os.Remove(targetFileName); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 拷贝文件而不是重命名
	srcFile, err := os.Open(tmpFname)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(targetFileName)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	if pkg.PackageType != PackageTypeExec {
		return nil
	}
	return os.Chmod(targetFileName, 0755)
}

/**
 *	在windows上设置PATH变量，让新安装的程序可以被执行
 */
func windowsSetPATH(installDir string) error {
	paths := os.Getenv("PATH")
	if !strings.Contains(paths, installDir) {
		newPath := fmt.Sprintf("%s;%s", paths, installDir)
		cmd := exec.Command("setx", "PATH", newPath)
		// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // 隐藏命令窗口
		if err := cmd.Run(); err != nil {
			return err
		}
		os.Setenv("PATH", newPath)
	}
	return nil
}

/**
 *	在linux上设置PATH变量，让新安装的程序可以被执行
 */
func linuxSetPATH(installDir string) error {
	currentPath := os.Getenv("PATH")
	// 检查是否已经包含该路径
	currentPathStr := strings.TrimSpace(currentPath)
	if strings.Contains(currentPathStr, installDir) {
		log.Println("The path is already in PATH.")
		return nil
	}
	// 将新路径添加到 PATH
	newPathStr := fmt.Sprintf("%s:%s", currentPathStr, installDir)
	err := os.Setenv("PATH", newPathStr)
	if err != nil {
		log.Printf("Failed to set PATH for current process: %v\n", err)
		return err
	}
	// 获取当前用户的主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v\n", err)
		return err
	}
	envLine := fmt.Sprintf("export PATH=$PATH:%s", installDir)

	bashrcPath := homeDir + "/.bashrc"
	// 检查是否已经包含该环境变量
	file, err := os.Open(bashrcPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to open ~/.bashrc: %v\n", err)
			return err
		}
		// 文件不存在，创建一个空文件
		file, err = os.Create(bashrcPath)
		if err != nil {
			log.Printf("Failed to create ~/.bashrc: %v\n", err)
			return err
		}
		file.Close()
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), envLine) {
				file.Close()
				log.Println("Environment variable already exists in ~/.bashrc.")
				return nil
			}
		}
		file.Close()
		if err := scanner.Err(); err != nil {
			log.Printf("Failed to read ~/.bashrc: %v\n", err)
			return err
		}
	}
	// 将环境变量追加到 ~/.bashrc 文件
	file, err = os.OpenFile(bashrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open ~/.bashrc for appending: %v\n", err)
		return err
	}
	defer file.Close()

	_, err = file.WriteString(envLine + "\n")
	if err != nil {
		log.Printf("Failed to write environment variable to ~/.bashrc: %v\n", err)
		return err
	}

	log.Println("Environment variable added to ~/.bashrc successfully.")
	return nil
}

/**
 *	安装包数据
 */
func installPackage(cfg UpgradeConfig, pkg PackageVersion, tmpFname string) error {
	if err := savePackageData(cfg, pkg, tmpFname); err != nil {
		return err
	}
	if pkg.PackageType != PackageTypeExec {
		return nil
	}
	if cfg.NoSetPath {
		return nil
	}
	if runtime.GOOS == "windows" {
		return windowsSetPATH(cfg.InstallDir)
	} else {
		return linuxSetPATH(cfg.InstallDir)
	}
}

/**
 *	移除指定名字的包
 *	@param {string} packageName - 要移除的包名称
 *	@param {string} baseDir - costrict数据所在的基路径，如果为空则使用默认路径
 *	@returns {error} 返回错误对象，成功时返回nil
 *	@description
 *	- 移除指定包的所有相关文件，包括包描述文件和安装的包文件
 *	- 首先读取包描述信息以确定需要删除的文件位置
 *	- 支持自定义baseDir，如果为空则使用默认的.costrict目录
 *	- 如果包不存在或已删除，不会报错
 *	@throws
 *	- 读取包描述文件失败
 *	- 删除包文件失败
 *	- 删除包描述文件失败
 *	@example
 *	err := RemovePackage("/home/xxx/.costrict", "my-package")
 *	if err != nil {
 *		log.Fatal(err)
 *	}
 */
func RemovePackage(baseDir string, packageName string) error {
	// 如果baseDir为空，使用默认路径
	if baseDir == "" {
		baseDir = getCostrictDir()
	}
	packageDir := filepath.Join(baseDir, "package")
	installDir := filepath.Join(baseDir, "bin")

	// 读取包描述文件
	packageFileName := filepath.Join(packageDir, fmt.Sprintf("%s.json", packageName))
	var pkg PackageVersion
	bytes, err := os.ReadFile(packageFileName)
	if err != nil {
		// 如果包描述文件不存在，认为包已移除，不报错
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("RemovePackage: read package file failed: %v", err)
	}

	// 解析包描述信息
	if err := json.Unmarshal(bytes, &pkg); err != nil {
		return fmt.Errorf("RemovePackage: unmarshal package info failed: %v", err)
	}

	// 删除包文件
	var targetFileName string
	if pkg.FileName == "" {
		targetFileName = filepath.Join(installDir, pkg.PackageName)
	} else {
		targetFileName = filepath.Join(installDir, pkg.FileName)
	}

	// 检查文件是否存在，如果存在则删除
	if _, err := os.Stat(targetFileName); err == nil {
		if err := os.Remove(targetFileName); err != nil {
			return fmt.Errorf("RemovePackage: remove package file '%s' failed: %v", targetFileName, err)
		}
		log.Printf("Package file '%s' removed successfully\n", targetFileName)
	}

	// 删除包描述文件
	if err := os.Remove(packageFileName); err != nil {
		return fmt.Errorf("RemovePackage: remove package description file '%s' failed: %v", packageFileName, err)
	}

	log.Printf("Package '%s' removed successfully\n", packageName)
	return nil
}
