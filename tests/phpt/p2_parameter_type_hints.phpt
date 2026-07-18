--TEST--
P2 parameter type hints accept PHP 5.3 arrays, classes, interfaces, self, parent, and nullable defaults
--FILE--
<?php
interface TypeContract {}
class TypeBase implements TypeContract {}
class TypeChild extends TypeBase {}

function type_sum(array $items) {
    return $items[0] + $items[1];
}

class TypeChecker {
    public function object(TypeBase $obj, TypeContract $contract, self $self = null) {
        return get_class($obj) . ':' . get_class($contract) . ':' . (is_object($self) ? get_class($self) : 'null');
    }
}

class TypeParent {}
class TypeParentChild extends TypeParent {
    public function accept(parent $value) {
        return get_class($value);
    }
}

$child = new TypeChild();
$checker = new TypeChecker();
echo type_sum(array(2, 3));
echo "\n";
echo $checker->object($child, $child, $checker);
echo "\n";
echo $checker->object($child, $child, null);
echo "\n";
echo (new TypeParentChild())->accept(new TypeParent());
?>
--EXPECT--
5
TypeChild:TypeChild:TypeChecker
TypeChild:TypeChild:null
TypeParent
