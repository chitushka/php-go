--TEST--
P3 ignored error policy suppresses type hint diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,warning,notice
php_src_go.error_halt=fatal,parse,warning,notice
--FILE--
<?php
class ExpectedHint {}
class OtherHint {}
function needs_object(ExpectedHint $value) {
    echo get_class($value);
}
needs_object(new OtherHint());
echo "\n";
echo 'after';
?>
--EXPECT--
OtherHint
after
