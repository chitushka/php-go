--TEST--
P3 strict error policy turns include warning into runtime error
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
$value = include 'p3_error_policy_strict_warning_no_such_file.php';
echo 'bad';
?>
--EXPECTF--
include failed for "p3_error_policy_strict_warning_no_such_file.php": open %s: %s
--EXIT_CODE--
1
