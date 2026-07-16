--TEST--
P2 abstract class can defer interface implementation
--FILE--
<?php
interface DeferredNamed {
    public function name();
}

abstract class DeferredAbstractUser implements DeferredNamed {
}

class DeferredUser extends DeferredAbstractUser {
    public function name() {
        return 'Ada';
    }
}

$user = new DeferredUser();
echo $user->name();
?>
--EXPECT--
Ada
