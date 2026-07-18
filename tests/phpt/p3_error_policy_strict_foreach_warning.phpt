--TEST--
P3 strict error policy halts on foreach non-array warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
foreach (1 as $value) {
    echo 'bad';
}
echo 'after';
?>
--EXPECT--
foreach source must be array
--EXIT_CODE--
1
