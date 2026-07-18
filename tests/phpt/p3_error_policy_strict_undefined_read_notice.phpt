--TEST--
P3 strict error policy halts on undefined read notices
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
echo $missing;
echo 'bad';
?>
--EXPECT--
undefined variable missing
--EXIT_CODE--
1
