--TEST--
P3 ignored warning policy suppresses OOP introspection diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$value = get_object_vars(1);
echo ($value === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
