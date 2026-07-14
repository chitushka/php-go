--TEST--
P1 magic constants
--FILE--
<?php
echo basename(__FILE__);
echo "\n";
echo basename(__DIR__);
echo "\n";
echo __LINE__;
echo "\n";

function magic_name() {
    echo __FUNCTION__;
}

class MagicBox {
    function show() {
        echo __FUNCTION__;
        echo ':';
        echo __METHOD__;
        echo ':';
        echo __CLASS__;
    }
}

magic_name();
echo "\n";
$box = new MagicBox();
$box->show();
?>
--EXPECT--
p1_magic_constants.phpt
phpt
6
magic_name
show:MagicBox::show:MagicBox
