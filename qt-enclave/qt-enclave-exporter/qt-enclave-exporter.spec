%global _bindir /usr/bin
%global debug_package %{nil}

Name: qt-enclave-exporter
Version: 1.0.0
Release: 1
Summary: qt-enclave collect qt vm cpu and memory usage
License: NA
Source0: %{name}.tar.gz

BuildRequires: golang > 1.21

%description
qt-enaclave collect qt vm cou and memory usage, and send to k8s 
through prometheus metrics interface.

%prep
cp %{SOURCE0} .

%build
tar zxvf %{SOURCE0}
cd %_topdir/BUILD/%{name}
export GOSUMDB=off
export GOPROXY=https://goproxy.cn,direct   
export GO111MODULE="on"
make

%install
install -d %{buildroot}%{_bindir}
# install binary
install -p -m 550 %_topdir/BUILD/%{name}/qt-enclave-exporter %{buildroot}%{_bindir}/qt-enclave-exporter

%files
%attr(0550,root,root) %{_bindir}/qt-enclave-exporter
%defattr(0640,root,root,0750)

%changelog
* Thu Mar 20 zhongjiawei <zhongjiawei1@huawei.com> -1.0.0-1
- init spec
