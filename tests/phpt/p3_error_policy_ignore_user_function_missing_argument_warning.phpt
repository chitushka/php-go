--TEST--
P3 ignored warning policy suppresses user function missing argument diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
function missing($value) {
    echo ($value === null ? 'null' : 'bad');
}
missing();
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
