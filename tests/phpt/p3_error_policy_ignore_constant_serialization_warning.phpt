--TEST--
P3 ignored warning policy suppresses constant and serialization diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$value = unserialize('broken');
echo ($value === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
false
after
