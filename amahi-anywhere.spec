Name:		amahi-anywhere
Version:	2.2
Release:	1
Summary:	Amahi Anywhere server

Group:          System Environment/Daemons
License:	Amahi Proprietary
Source:         %{name}-%{version}.tar.gz
BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:	golang
Requires:	mailcap

%define debug_package %{nil}

%description
Amahi Anywhere server

%prep
%setup -q

%build
make %{?_smp_mflags} build-%{BUILD_TYPE}

%install
%{__mkdir} -p %{buildroot}%{_bindir} %{buildroot}%{_unitdir}
%{__install} -m 755 bin/fs %{buildroot}%{_bindir}/amahi-anywhere
%{__install} -D -m 0644 -p amahi-anywhere.service %{buildroot}%{_unitdir}/

%post
%systemd_post amahi-anywhere.service
/usr/bin/systemctl enable amahi-anywhere
/usr/bin/systemctl start amahi-anywhere

%preun
%systemd_preun amahi-anywhere.service
/usr/bin/systemctl daemon-reload > /dev/null 2>&1 || :

%postun
%systemd_postun_with_restart amahi-anywhere.service

%files
%doc
%{_bindir}/amahi-anywhere
%{_unitdir}/amahi-anywhere.service

%changelog

