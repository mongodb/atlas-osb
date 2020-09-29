#!/bin/bash
assert_equal() {
  actual=$1
  expected=$2
  if [[ "${actual}" != "${expected}" ]]; then
    echo "Assert failed: ${actual} is not equal ${expected}. Must be equal"
    exit 1
  fi
  echo "Assert passed: ${actual} is equal ${expected}"
}

assert_not_equal() {
  actual=$1
  expected=$2
  if [[ "${actual}" == "${expected}" ]]; then
    echo "Assert failed: ${actual} is equal ${expected}. Must be not equal"
    exit 1
  fi
  echo "Assert passed: ${actual} is not equal ${expected}"
}