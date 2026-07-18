--TEST--
P3 error policy can report type hint errors without halting
--INI--
php_src_go.error_halt=fatal,parse,warning,notice
--FILE--
<?php
function needs_array(array $items) {
    echo is_string($items) ? 'string' : 'bad';
}
needs_array('no');
echo "\n";
echo 'after';
?>
--EXPECT--
string
after
