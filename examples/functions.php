<?php
echo sum_values(array(1, 2, 3));

function sum_values($items) {
    $sum = 0;
    foreach ($items as $value) {
        $sum = $sum + $value;
    }
    return $sum;
}
