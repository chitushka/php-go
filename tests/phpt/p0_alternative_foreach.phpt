--TEST--
P0 alternative foreach syntax
--FILE--
<?php
$items = array('a' => 1, 'b' => 2, 'c' => 3);
foreach ($items as $key => $value):
    if ($key === 'b'):
        continue;
    endif;
    echo $key;
    echo $value;
endforeach;
echo ':';
foreach (array(4, 5) as $value):
    echo $value;
endforeach;
?>
--EXPECT--
a1c3:45
