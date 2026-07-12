--TEST--
interfaces and implements
--FILE--
<?php
interface Named {
    public function name();
}

interface Labeled extends Named {
    public function label();
}

class User implements Labeled {
    public function name() {
        return 'Ada';
    }

    public function label() {
        return 'user';
    }
}

$user = new User();
echo $user->name();
echo ':';
echo $user->label();
--EXPECT--
Ada:user
