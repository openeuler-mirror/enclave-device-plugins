%global _bindir /usr/bin
%global debug_package %{nil}
 
Name: qt-device-plugin
Version: 1.0.0
Release: 1
Summary: qt-enclave device plugin manages device qtbox_service0.
License: NA
Source0: %{name}.tar.gz
 
BuildRequires: golang > 1.21
 
%description
qt-enaclave device plugin gives your pods and containers the ability to access the qtbox_service0.
 
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
install -p -m 550 %_topdir/BUILD/%{name}/qt-enclave-k8s-device-plugin %{buildroot}%{_bindir}/qt-enclave-k8s-device-plugin
 
%files
%attr(0550,root,root) %{_bindir}/qt-enclave-k8s-device-plugin
%defattr(0640,root,root,0750)
 
%changelog
* Thu Mar 20 liuxu <liuxu156@huawei.com> -1.0.0-1
- init spec