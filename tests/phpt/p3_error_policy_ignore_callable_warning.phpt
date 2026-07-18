--TEST--
P3 ignored warning policy suppresses callable builtin diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$value = call_user_func(array('not-enough'));
echo ($value === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
