--TEST--
P3 strict error policy halts on isset and empty warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
isset();
echo 'bad';
?>
--EXPECT--
isset expects at least 1 argument, got 0
--EXIT_CODE--
1
