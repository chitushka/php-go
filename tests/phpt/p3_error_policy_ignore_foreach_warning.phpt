--TEST--
P3 ignored warning policy suppresses foreach non-array diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
foreach ('not-array' as $value) {
    echo 'bad';
}
echo 'after';
?>
--EXPECT--
after
