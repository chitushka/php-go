--TEST--
P3 ignored warning policy suppresses preg_match capture diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$value = preg_match('/[/', 'x', $matches);
echo ($value === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
false
after
