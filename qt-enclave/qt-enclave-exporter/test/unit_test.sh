#!/bin/bash
  
# Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
# Description: This shell script is used to do unit test.
# Create: 2025-04-01

exit_flag=0

function cleanup_fun()
{
    rm -rf /var/log/qlog
    kill -9 $(pidof qt-enclave-exporter)
    rm -rf test.log
}

function test_fun()
{
    # test case: no qt log file
    /usr/bin/qt-enclave-exporter -p 9112 > test.log 2>&1 &
    sleep 2
    num=`cat test.log |grep "no such file or directory" |wc -l`
    if [ $num -eq 0 ];then
        echo "test log file failed"
        exit_flag=$(($exit_flag+1))
    fi
    # restart qt-enclave-exporter, and create qt log file
    kill -9 $(pidof qt-enclave-exporter)
    rm -rf test.log
    mkdir -p /var/log/qlog
    touch /var/log/qlog/resource.log
    /usr/bin/qt-enclave-exporter -p 9112 > test.log 2>&1 &
    sleep 2

    # test case: check whether the qlog data is correct
    echo "2025-02-21T07:53:13.917605838+0000: CpuUsage: 1.8%, MemTotal: 994740 kB, MemFree: 830708 kB, MemAvailable: 765508 kB " >> /var/log/qlog/resource.log 
    cpu_usage=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_cpu_usage_percent |grep -v "#" | awk '{print $2}'`
    total_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_total |grep -v "#" | awk '{print $2}'`
    free_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_free |grep -v "#" | awk '{print $2}'`
    ava_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_available |grep -v "#" | awk '{print $2}'`
    if [ $free_mem != "830708" ] || [ $total_mem != "994740" ] || [ $cpu_usage != "1.8" ] || [ $ava_mem != "765508" ];then
        echo "get qlog and decode wrong"
        exit_flag=$(($exit_flag+1))
    fi

    # test case: check wrong qlog format
    echo "111111" >> /var/log/qlog/resource.log
    num1=`cat test.log |grep "log format is not right" |wc -l`
    if [ $num1 -ne 1 ];then
        echo "test wrong log format failed"
        exit_flag=$(($exit_flag+1))
    fi

    # test case: check the same qlog
    echo "2025-02-21T07:53:13.917605838+0000: CpuUsage: 1.8%, MemTotal: 994740 kB, MemFree: 830708 kB, MemAvailable: 765508 kB " >> /var/log/qlog/resource.log
    num2=`cat test.log |grep "qlog is not update" |wc -l`
    if [ $num2 -ne 1 ];then
        echo "test same log failed"
        exit_flag=$(($exit_flag+1))
    fi

    # test case: test qlog rotation
    mv /var/log/qlog/resource.log /var/log/qlog/resource1.log
    sleep 2
    num3=`cat test.log |grep "no such file or directory" |wc -l`
    if [ $num3 -eq 0 ];then
        echo "test log file failed"
        exit_flag=$(($exit_flag+1))
    fi
    touch /var/log/qlog/resource.log
    sleep 2

    #test case: check whether the qlog data is correct when qlog update
    echo "2025-02-21T07:53:13.917605838+0000: CpuUsage: 12.8%, MemTotal: 994740 kB, MemFree: 88886 kB, MemAvailable: 666668 kB " >> /var/log/qlog/resource.log
    cpu_usage=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_cpu_usage_percent |grep -v "#" | awk '{print $2}'`
    total_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_total |grep -v "#" | awk '{print $2}'`
    free_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_free |grep -v "#" | awk '{print $2}'`
    ava_mem=`curl -s http://127.0.0.1:9112/metrics |grep qingtian_memory_available |grep -v "#" | awk '{print $2}'`
    if [ $free_mem != "88886" ] || [ $total_mem != "994740" ] || [ $cpu_usage != "12.8" ] || [ $ava_mem != "666668" ];then
        echo "after restart, get qlog and decode wrong"
        exit_flag=$(($exit_flag+1))
    fi
}

cleanup_fun
test_fun
cleanup_fun
exit $exit_flag
