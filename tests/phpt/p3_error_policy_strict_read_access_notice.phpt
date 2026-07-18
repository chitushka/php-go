--TEST--
P3 strict error policy halts on scalar read access notices
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
$x = 1;
$value = $x['key'];
echo 'bad';
?>
--EXPECT--
cannot use index on 1
--EXIT_CODE--
1
