--TEST--
P2 class_alias autoload and lookup
--FILE--
<?php
function __autoload($class) {
    include 'p2_class_alias_lib.php';
}

echo class_alias('AliasSource', 'AliasTarget') ? 'alias' : 'no';
echo ':';
echo class_exists('aliastarget', false) ? 'exists' : 'missing';
echo ':';

$obj = new AliasTarget();
echo get_class($obj);
echo ':';
echo $obj->name();
echo ':';
echo AliasTarget::LABEL;
echo ':';
echo AliasTarget::bump();
echo ':';
echo is_a($obj, 'AliasTarget') ? 'isa' : 'no';
echo ':';

$classes = get_declared_classes();
echo in_array('AliasTarget', $classes) ? 'declared' : 'missing';
echo ':';
echo class_alias('AliasSource', 'AliasTarget') ? 'again' : 'duplicate';
--EXPECT--
alias:exists:AliasSource:object:source:3:isa:declared:duplicate
