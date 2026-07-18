--TEST--
P3 ignored notice policy suppresses undefined read diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,warning
php_src_go.error_halt=fatal,parse,error,warning
--FILE--
<?php
$arr = array();
$value = $arr['missing'];
echo ($value === null ? 'null' : 'bad');
echo "\n";
if (!isset($arr['missing']) && empty($arr['missing'])) {
    echo 'constructs';
}
echo "\n";
echo 'after';
?>
--EXPECT--
null
constructs
after
