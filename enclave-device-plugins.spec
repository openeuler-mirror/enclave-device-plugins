%global _bindir /usr/bin
%global debug_package %{nil}

Name: enclave-device-plugins
Version: 1.0.0
Release: 1
Summary: enclave-device-plugins provides a set of daemonsets for kubernetes to manage enclave confidential containers
License: NA
Source0: %{name}.tar.gz

BuildRequires: golang > 1.21

%description
qt-enaclave device plugin gives your pods and containers the ability to access the qtbox_service0.
qt-enaclave-export collects qt vm cpu and memory usage, and sends to k8s through prometheus metrics interface.

%prep
cp %{SOURCE0} .

%build
tar zxvf %{SOURCE0}
export GOSUMDB=off
export GOPROXY=https://goproxy.cn,direct   
export GO111MODULE="on"
cd %_topdir/BUILD/%{name}/qt-enclave/qt-enclave-device-plugin
make
cd %_topdir/BUILD/%{name}/qt-enclave/qt-enclave-exporter
make

%install
install -d %{buildroot}%{_bindir}
# install binary
install -p -m 550 %_topdir/BUILD/%{name}/qt-enclave/qt-enclave-exporter/qt-enclave-exporter %{buildroot}%{_bindir}/qt-enclave-exporter
install -p -m 550 %_topdir/BUILD/%{name}/qt-enclave/qt-enclave-device-plugin/qt-enclave-k8s-device-plugin %{buildroot}%{_bindir}/qt-enclave-k8s-device-plugin

%files
%attr(0550,root,root) %{_bindir}/qt-enclave-exporter
%attr(0550,root,root) %{_bindir}/qt-enclave-k8s-device-plugin
%defattr(0640,root,root,0750)

%changelog
* Mon Apr 14 zhongjiawei <zhongjiawei1@huawei.com> -1.0.0-1
- init spec
