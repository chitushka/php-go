--TEST--
P0 variable variables
--FILE--
<?php
$name = 'target';
$$name = 'value';
echo $target;
echo ':';
$target = 'changed';
echo $$name;
echo ':';
$$name .= '-ok';
echo $target;
echo ':';
$arrayName = 'items';
$$arrayName['nested'][] = 'x';
echo $items['nested'][0];
echo ':';
if (isset($$name)) {
    echo 'isset';
}
unset($$name);
if (!isset($target)) {
    echo ':unset';
}
?>
--EXPECT--
value:changed:changed-ok:x:isset:unset
