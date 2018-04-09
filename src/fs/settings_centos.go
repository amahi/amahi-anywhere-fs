// +build centos,linux

/*
 * Copyright (c) 2013-2018 Amahi
 *
 * This file is part of Amahi.
 *
 * Amahi is free software released under the GNU GPL v3 license.
 * See the LICENSE file accompanying this distribution.
 */

package main

// Path for Centos

const MYSQL_CREDENTIALS = "amahihda:AmahiHDARulez@unix(/var/lib/mysql/mysql.sock)/hda_production?parseTime=true"
const SQL_SELECT_SHARES = "SELECT name, updated_at, path FROM shares WHERE visible = 1 ORDER BY name ASC"
const SQL_SELECT_APPS = "SELECT webapps.name, apps.name, apps.logo_url FROM webapps LEFT OUTER JOIN apps on apps.webapp_id = webapps.id ORDER BY apps.name ASC"

const METADATA_FILE = "/tmp/aamd.db"

const PLATFORM = "centos"

const PID_FILE = "/run/amahi-anywhere.pid"
