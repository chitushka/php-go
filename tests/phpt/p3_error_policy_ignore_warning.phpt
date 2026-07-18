--TEST--
P3 error policy can ignore include warnings and continue
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$value = include 'p3_error_policy_ignore_warning_no_such_file.php';
echo ($value === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
false
after
