// +build ubuntu,linux

/*
 * Copyright (c) 2013-2018 Amahi
 *
 * This file is part of Amahi.
 *
 * Amahi is free software released under the GNU GPL v3 license.
 * See the LICENSE file accompanying this distribution.
 */

package main

// Path for Ubuntu

const MYSQL_CREDENTIALS = "amahihda:AmahiHDARulez@unix(/var/run/mysqld/mysqld.sock)/hda_production?parseTime=true"
const SQL_SELECT_SHARES = "SELECT comment, updated_at, path, tags FROM shares WHERE visible = 1 ORDER BY comment ASC"
const SQL_SELECT_APPS = "SELECT webapps.name, apps.name, apps.logo_url FROM webapps LEFT OUTER JOIN apps on apps.webapp_id = webapps.id ORDER BY apps.name ASC"

const METADATA_FILE = "/tmp/aamd.db"

const PLATFORM = "ubuntu"

const PID_FILE = "/run/amahi-anywhere.pid"
