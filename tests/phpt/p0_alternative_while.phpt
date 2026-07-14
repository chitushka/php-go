--TEST--
P0 alternative while syntax
--FILE--
<?php
$i = 0;
while ($i < 5):
    $i++;
    if ($i === 2):
        continue;
    endif;
    if ($i === 4):
        break;
    endif;
    echo $i;
endwhile;
echo ':';
echo $i;
?>
--EXPECT--
13:4
