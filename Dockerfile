#
# Dockerfile - Copyright (c) 2023-2026 - Olivier Poncet
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
# build stage
# ----------------------------------------------------------------------------

FROM golang:1.27-alpine3.23 AS build

WORKDIR /usr/src/timesheet

COPY . .

RUN apk add --no-cache tzdata \
 && sh .ci/make-build.sh \
 && DESTDIR=/stage sh .ci/make-install.sh \
 && install -m 755 .ci/timesheet-entrypoint.sh /stage/usr/bin/timesheet-entrypoint.sh \
 && /stage/usr/bin/timesheet config set server.listen 0.0.0.0:8080 server.trusted_proxies '' --config=/stage/etc/timesheet/timesheet.json

# ----------------------------------------------------------------------------
# runtime stage
# ----------------------------------------------------------------------------

FROM alpine:3.22

RUN apk add --no-cache tzdata \
 && addgroup -S -g 10001 timesheet \
 && adduser -S -D -H -u 10001 -G timesheet -h /var/lib/timesheet -s /sbin/nologin timesheet \
 && install -d -o timesheet -g timesheet -m 0700 /var/lib/timesheet \
 && install -d -o timesheet -g timesheet -m 0755 /etc/timesheet

COPY --from=build /stage/usr/ /usr/
COPY --from=build --chown=timesheet:timesheet /stage/etc/timesheet/ /etc/timesheet/

USER timesheet:timesheet
WORKDIR /var/lib/timesheet
VOLUME ["/var/lib/timesheet", "/etc/timesheet"]
EXPOSE 8080

ENTRYPOINT ["/usr/bin/timesheet-entrypoint.sh"]
CMD ["run"]

# ----------------------------------------------------------------------------
# End-Of-File
# ----------------------------------------------------------------------------
