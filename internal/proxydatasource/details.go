// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proxydatasource implements an internal.DataSource backed solely by a
// proxy instance.
package proxydatasource

import (
	"context"
	"fmt"
	"path"
	"strings"

	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/derrors"
	"golang.org/x/pkgsite/internal/licenses"
	"golang.org/x/pkgsite/internal/proxy"
)

// GetUnit returns information about a directory at a path.
func (ds *DataSource) GetUnit(ctx context.Context, pi *internal.PathInfo, field internal.FieldSet) (_ *internal.Unit, err error) {
	defer derrors.Wrap(&err, "GetUnit(%q, %q, %q)", pi.Path, pi.ModulePath, pi.Version)
	return ds.getUnit(ctx, pi.Path, pi.ModulePath, pi.Version)
}

// GetDirectoryMeta returns information about a directory at a path.
func (ds *DataSource) GetDirectoryMeta(ctx context.Context, fullPath, modulePath, version string) (_ *internal.DirectoryMeta, err error) {
	defer derrors.Wrap(&err, "GetDirectoryMeta(%q, %q, %q)", fullPath, modulePath, version)
	d, err := ds.getUnit(ctx, fullPath, modulePath, version)
	if err != nil {
		return nil, err
	}
	return &d.DirectoryMeta, nil
}

// GetLicenses return licenses at path for the given module path and version.
func (ds *DataSource) GetLicenses(ctx context.Context, fullPath, modulePath, resolvedVersion string) (_ []*licenses.License, err error) {
	defer derrors.Wrap(&err, "GetLicenses(%q, %q, %q)", fullPath, modulePath, resolvedVersion)
	v, err := ds.getModule(ctx, modulePath, resolvedVersion)
	if err != nil {
		return nil, err
	}

	var lics []*licenses.License

	// ds.getModule() returns all licenses for the module version. We need to
	// filter the licenses that applies to the specified fullPath, i.e.
	// A license in the current or any parent directory of the specified
	// fullPath applies to it.
	for _, license := range v.Licenses {
		licensePath := path.Join(modulePath, path.Dir(license.FilePath))
		if strings.HasPrefix(fullPath, licensePath) {
			lics = append(lics, license)
		}
	}

	if len(lics) == 0 {
		return nil, fmt.Errorf("path %s is missing from module %s: %w", fullPath, modulePath, derrors.NotFound)
	}
	return lics, nil
}

// GetModuleInfo returns the ModuleInfo as fetched from the proxy for module
// version specified by modulePath and version.
func (ds *DataSource) GetModuleInfo(ctx context.Context, modulePath, version string) (_ *internal.ModuleInfo, err error) {
	defer derrors.Wrap(&err, "GetModuleInfo(%q, %q)", modulePath, version)
	m, err := ds.getModule(ctx, modulePath, version)
	if err != nil {
		return nil, err
	}
	return &m.ModuleInfo, nil
}

// GetPathInfo returns information about the given path.
func (ds *DataSource) GetPathInfo(ctx context.Context, path, inModulePath, inVersion string) (_ *internal.PathInfo, err error) {
	defer derrors.Wrap(&err, "GetPathInfo(%q, %q, %q)", path, inModulePath, inVersion)

	var info *proxy.VersionInfo
	if inModulePath == internal.UnknownModulePath {
		inModulePath, info, err = ds.findModule(ctx, path, inVersion)
		if err != nil {
			return nil, err
		}
		inVersion = info.Version
	}
	m, err := ds.getModule(ctx, inModulePath, inVersion)
	if err != nil {
		return nil, err
	}
	pi := &internal.PathInfo{
		Path:       path,
		ModulePath: inModulePath,
		Version:    inVersion,
	}
	for _, d := range m.Units {
		if d.Path == path {
			pi.Name = d.Name
			pi.IsRedistributable = d.IsRedistributable
			break
		}
	}
	return pi, nil
}

// GetExperiments is unimplemented.
func (*DataSource) GetExperiments(ctx context.Context) ([]*internal.Experiment, error) {
	return nil, nil
}
