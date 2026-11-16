#!/bin/sh
#
# ci-build-package.sh - Copyright (c) 2023-2026 - Olivier Poncet
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 2 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

# ----------------------------------------------------------------------------
# debug
# ----------------------------------------------------------------------------

set -x

# ----------------------------------------------------------------------------
# build directories
# ----------------------------------------------------------------------------

arg_prefix="$(pwd)"
arg_parent="$(dirname "${arg_prefix}")"
arg_dist_dir="${arg_prefix}/_dist"
arg_basename='timesheet'

# ----------------------------------------------------------------------------
# bump package version (ci only)
# ----------------------------------------------------------------------------

if [ "${CI:-not-set}" != 'not-set' ]
then
    pkg_version="$(dpkg-parsechangelog --show-field Version)+$(date '+%Y%m%d%H%M%S')"
    pkg_changelog='debian/changelog'
    pkg_tmpfile='debian/changelog.tmp'
    {
        echo "${arg_basename} (${pkg_version}) stable; urgency=medium"
        echo ''
        echo '  * Automated CI build.'
        echo ''
        echo " -- Olivier Poncet <ponceto@free.fr>  $(date -R)"
        echo ''
        cat "${pkg_changelog}"
    } > "${pkg_tmpfile}"                                             || exit 1
    mv "${pkg_tmpfile}" "${pkg_changelog}"                           || exit 1
fi

# ----------------------------------------------------------------------------
# check the toolchain
# ----------------------------------------------------------------------------

go version                                                           || exit 1

# ----------------------------------------------------------------------------
# build package
# ----------------------------------------------------------------------------

dpkg-buildpackage --build=any,all --no-sign                          || exit 1

# ----------------------------------------------------------------------------
# move the packages to the dist directory
# ----------------------------------------------------------------------------

mkdir -p "${arg_dist_dir}"                                           || exit 1
mv "${arg_parent}/${arg_basename}"_* "${arg_dist_dir}"               || exit 1
mv "${arg_parent}/${arg_basename}"-dbgsym_* "${arg_dist_dir}"        || exit 1

# ----------------------------------------------------------------------------
# End-Of-File
# ----------------------------------------------------------------------------
