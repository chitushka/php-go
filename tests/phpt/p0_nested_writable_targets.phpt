--TEST--
P0 nested writable array targets
--FILE--
<?php
$items = array();
$items['user']['name'] = 'Ada';
$items['user']['roles'][] = 'admin';
$items['user']['roles'][] = 'editor';
$items['counts']['visits'] = 1;
$items['counts']['visits'] += 2;
echo $items['user']['name'];
echo ':';
echo $items['user']['roles'][0];
echo ':';
echo $items['user']['roles'][1];
echo ':';
echo $items['counts']['visits'];
?>
--EXPECT--
Ada:admin:editor:3
